package credhelper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeHelperOnPath writes a fake docker-credential-<name> shell script onto
// PATH that emits the given username and secret as a docker-credential-helper
// Get response. Returns the helper name.
func writeHelperOnPath(t *testing.T, username, secret string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell helper not portable to windows")
	}

	binDir := t.TempDir()
	name := "credhelper-test"
	script := "#!/bin/sh\ncat > /dev/null\n" +
		`printf '{"ServerURL":"","Username":"` + username + `","Secret":"` + secret + `"}\n'` + "\n"
	helperPath := filepath.Join(binDir, "docker-credential-"+name)
	if err := os.WriteFile(helperPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	orig := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+orig)
	return name
}

// writeDockerConfig writes a docker config.json in a temp dir, points
// DOCKER_CONFIG at it, and maps the given host to the given credHelper name.
// Setting DOCKER_CONFIG explicitly sidesteps homedir.Dir caching between tests.
func writeDockerConfig(t *testing.T, host, helperName string) {
	t.Helper()

	dockerDir := t.TempDir()
	t.Setenv("DOCKER_CONFIG", dockerDir)

	cfg := map[string]any{
		"credHelpers": map[string]string{
			host: helperName,
		},
	}
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), body, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// TestRegistryHostsStaticBearerSentinel verifies that a docker-credential
// helper returning the "<bearer>" sentinel username produces a RegistryHost
// with a static "Authorization: Bearer <Secret>" header and no Authorizer —
// so the challenge-response auth flow is skipped entirely.
func TestRegistryHostsStaticBearerSentinel(t *testing.T) {
	const host = "registry.example.com"
	const token = "tok-abc-123"

	helperName := writeHelperOnPath(t, staticBearerSentinel, token)
	writeDockerConfig(t, host, helperName)

	hosts, err := RegistryHostsFromDockerConfig()(host)
	if err != nil {
		t.Fatalf("RegistryHostsFromDockerConfig: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("want 1 host, got %d", len(hosts))
	}
	h := hosts[0]

	if got := h.Header.Get("Authorization"); got != "Bearer "+token {
		t.Errorf("Header[Authorization] = %q, want %q", got, "Bearer "+token)
	}
	if h.Authorizer != nil {
		t.Errorf("Authorizer must be nil in static-Bearer mode; got %T", h.Authorizer)
	}
}

// Note: the non-sentinel (challenge-response) branch is not exercised here
// because seedAuthHeaders hardcodes https:// and does a network round-trip,
// which is awkward to unit test. That code path is unchanged by this patch
// and is covered by existing rules_oci integration usage.
