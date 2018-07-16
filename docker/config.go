package docker

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const apiVersion = "1.37"

// Config is the structure that stores the configuration to talk to a
// Docker API compatible host.
type Config struct {
	Host     string
	Ca       string
	Cert     string
	Key      string
	CertPath string
}

// NewClient returns a new Docker client.
func (c *Config) NewClient() (*client.Client, error) {
	if c.Ca != "" || c.Cert != "" || c.Key != "" {
		if c.Ca == "" || c.Cert == "" || c.Key == "" {
			return nil, fmt.Errorf("ca_material, cert_material, and key_material must be specified")
		}

		if c.CertPath != "" {
			return nil, fmt.Errorf("cert_path must not be specified")
		}

		return client.NewClientWithOpts(
			client.WithHost(c.Host),
			client.WithTLSClientConfig(c.Ca, c.Cert, c.Key),
			client.WithVersion(apiVersion),
		)
	}

	if c.CertPath != "" {
		// If there is cert information, load it and use it.
		ca := filepath.Join(c.CertPath, "ca.pem")
		cert := filepath.Join(c.CertPath, "cert.pem")
		key := filepath.Join(c.CertPath, "key.pem")
		return client.NewClientWithOpts(
			client.WithHost(c.Host),
			client.WithTLSClientConfig(ca, cert, key),
			client.WithVersion(apiVersion),
		)
	}

	// If there is no cert information, then just return the direct client
	return client.NewClientWithOpts(
		client.WithHost(c.Host),
		client.WithVersion(apiVersion),
	)
}

// Data structure for holding data that we fetch from Docker.
type Data struct {
	DockerImages map[string]*types.ImageSummary
}

// ProviderConfig for the custom registry provider
type ProviderConfig struct {
	DockerClient *client.Client
	AuthConfigs  *AuthConfigs
}

// The registry address can be referenced in various places (registry auth, docker config file, image name)
// with or without the http(s):// prefix; this function is used to standardize the inputs
func normalizeRegistryAddress(address string) string {
	if !strings.HasPrefix(address, "https://") && !strings.HasPrefix(address, "http://") {
		return "https://" + address
	}
	return address
}
