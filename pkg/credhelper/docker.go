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

		helperName, ok := cfg.CredentialHelpers[host]
		if !ok {
			return nil, nil
		}

		registryHost := docker.RegistryHost{
			Host:         host,
			Scheme:       "https",
			Path:         "/v2",
			Capabilities: docker.HostCapabilityPull | docker.HostCapabilityResolve | docker.HostCapabilityPush,
		}

		registryHost.Authorizer = docker.NewDockerAuthorizer(docker.WithAuthCreds(func(host string) (string, string, error) {
			p := helperclient.NewShellProgramFunc(fmt.Sprintf("docker-credential-%s", helperName))

			creds, err := helperclient.Get(p, fmt.Sprintf("%s://%s", registryHost.Scheme, registryHost.Host))
			if err != nil {
				return "", "", err
			}

			return creds.Username, creds.Secret, nil
		}))

		err = seedAuthHeaders(registryHost)
		if err != nil {
			return nil, err
		}

		return []docker.RegistryHost{registryHost}, nil
	}
}
