package credhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/remotes/docker"
	helperclient "github.com/docker/docker-credential-helpers/client"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
)

const (
	DockerConfigEnv         = "DOCKER_CONFIG"
	DefaultDockerConfigName = "config.json"
)

func GetConfigDir() (string, error) {
	var base string
	var err error
	if val := os.Getenv(DockerConfigEnv); val != "" {
		base = val
	} else {
		base, err = homedir.Dir()
		if err != nil {
			return "", err
		}

		base = filepath.Join(base, ".docker")
	}

	return filepath.Join(base, DefaultDockerConfigName), nil
}

type DockerConfig struct {
	CredentialHelpers map[string]string `json:"credHelpers"`
}

func ReadDockerConfig(re io.Reader) (DockerConfig, error) {
	var cfg DockerConfig

	err := json.NewDecoder(re).Decode(&cfg)
	if err != nil {
		return DockerConfig{}, err
	}

	if cfg.CredentialHelpers == nil {
		cfg.CredentialHelpers = make(map[string]string)
	}

	return cfg, nil
}

func ReadHostDockerConfig() (DockerConfig, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return DockerConfig{}, err
	}

	f, err := os.Open(dir)
	if err != nil {
		return DockerConfig{}, err
	}
	defer f.Close()

	return ReadDockerConfig(f)
}

func seedAuthHeaders(host docker.RegistryHost) error {
	if host.Authorizer == nil {
		return nil
	}

	cli := http.DefaultClient
	if host.Client != nil {
		cli = host.Client
	}

	v2URL := fmt.Sprintf("%s://%s%s/", host.Scheme, host.Host, host.Path)
	resp, err := cli.Get(v2URL)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		log.WithField("url", v2URL).Debug("seeding authorization headers")
		err = host.Authorizer.AddResponses(context.Background(), []*http.Response{resp})
		if err != nil && !errdefs.IsNotImplemented(err) {
			return err
		}
	}

	return nil
}

// bearerAuthFixTransport intercepts requests where containerd has set
// Authorization: Basic base64("Bearer:<JWT>") (from a credential helper returning
// Username="Bearer") and converts them to Authorization: Bearer <JWT>.
// This allows the token endpoint to issue a proper scope-specific token, which
// containerd then uses for all subsequent registry requests (manifests and blobs).
type bearerAuthFixTransport struct {
	token string
}

func (t *bearerAuthFixTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if username, _, ok := clone.BasicAuth(); ok && username == "Bearer" {
		clone.Header.Set("Authorization", "Bearer "+t.token)
	}
	return http.DefaultTransport.RoundTrip(clone)
}

func RegistryHostsFromDockerConfig() docker.RegistryHosts {
	return func(host string) ([]docker.RegistryHost, error) {
		// FIXME This should be cached somewhere
		cfg, err := ReadHostDockerConfig()
		if err != nil {
			return nil, err
		}

		// Don't error if the file doesn't exist
		if os.IsNotExist(err) {
			return nil, nil
		}

		registryHost := docker.RegistryHost{
			Host:         host,
			Scheme:       "https",
			Path:         "/v2",
			Capabilities: docker.HostCapabilityPull | docker.HostCapabilityResolve | docker.HostCapabilityPush,
		}

		helperName, ok := cfg.CredentialHelpers[host]
		if !ok {
			// If no credential helper is specified, fall back on the default behavior.
			registryHost.Authorizer = docker.NewDockerAuthorizer()
			return []docker.RegistryHost{registryHost}, nil
		}

		p := helperclient.NewShellProgramFunc(fmt.Sprintf("docker-credential-%s", helperName))
		creds, err := helperclient.Get(p, fmt.Sprintf("%s://%s", registryHost.Scheme, registryHost.Host))
		if err != nil {
			return nil, err
		}

		if creds.Username == "Bearer" {
			// The credential helper returned a pre-issued Bearer token (Username=="Bearer",
			// Secret==<JWT>). containerd's WithAuthCreds flow would construct
			// Authorization: Basic base64("Bearer:<JWT>") for the token endpoint, which
			// the registry rejects with 400 Bad Request.
			//
			// Fix: use a custom transport that converts Basic("Bearer", <JWT>) →
			// Bearer <JWT> on the wire. This lets the token endpoint issue a proper
			// scope-specific token, which containerd then uses for all registry requests
			// (manifests and blobs).
			customClient := &http.Client{
				Transport: &bearerAuthFixTransport{token: creds.Secret},
			}
			registryHost.Client = customClient
			registryHost.Authorizer = docker.NewDockerAuthorizer(
				docker.WithAuthCreds(func(host string) (string, string, error) {
					return creds.Username, creds.Secret, nil
				}),
				docker.WithAuthClient(customClient),
			)

			err = seedAuthHeaders(registryHost)
			if err != nil {
				return nil, err
			}

			return []docker.RegistryHost{registryHost}, nil
		}

		registryHost.Authorizer = docker.NewDockerAuthorizer(docker.WithAuthCreds(func(host string) (string, string, error) {
			return creds.Username, creds.Secret, nil
		}))

		err = seedAuthHeaders(registryHost)
		if err != nil {
			return nil, err
		}

		return []docker.RegistryHost{registryHost}, nil
	}
}
