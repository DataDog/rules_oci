package credhelper

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/docker-credential-helpers/client"
	"github.com/mitchellh/go-homedir"
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

func HelperToRegistryHost(reg, helper string) docker.RegistryHosts {
	return func(host string) ([]docker.RegistryHost, error) {
        if host != reg {
            return nil, nil
        }

		return []docker.RegistryHost{
			{
				Host:         reg,
				Scheme:       "https",
				Path:         "/v2",
				Capabilities: docker.HostCapabilityPull | docker.HostCapabilityResolve | docker.HostCapabilityPush,
				Authorizer: docker.NewDockerAuthorizer(docker.WithAuthCreds(func(host string) (string, string, error) {
					p := helperclient.NewShellProgramFunc(fmt.Sprintf("docker-credential-%s", helper))

					creds, err := helperclient.Get(p, fmt.Sprintf("%s://%s"))
					if err != nil {
						return "", "", err
					}

					return creds.Username, creds.Secret, nil
				})),
			},
		}, nil
	}
}

func GetHostsFromDockerConfig() (docker.RegistryHosts, error) {
	cfg, err := ReadHostDockerConfig()
	if err != nil {
		return nil, err
	}

	var hosts []docker.RegistryHosts
	for name, helper := range cfg.CredentialHelpers {
		hosts = append(hosts, HelperToRegistryHost(name, helper))
	}

    // Support for Docker Hub
    hosts = append(hosts, docker.ConfigureDefaultRegistries())

	return docker.Registries(hosts...), nil
}
