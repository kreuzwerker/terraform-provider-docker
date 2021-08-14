package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Config is the structure that stores the configuration to talk to a
// Docker API compatible host.
type Config struct {
	Host     string
	Ca       string
	Cert     string
	Key      string
	CertPath string
}

func NewConfig(d *schema.ResourceData) *Config {
	return &Config{
		Host:     d.Get("host").(string),
		Ca:       d.Get("ca_material").(string),
		Cert:     d.Get("cert_material").(string),
		Key:      d.Get("key_material").(string),
		CertPath: d.Get("cert_path").(string),
	}
}

// buildHTTPClientFromBytes builds the http client from bytes (content of the files)
func buildHTTPClientFromBytes(caPEMCert, certPEMBlock, keyPEMBlock []byte) (*http.Client, error) {
	tlsConfig := &tls.Config{}
	if certPEMBlock != nil && keyPEMBlock != nil {
		tlsCert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{tlsCert}
	}

	if len(caPEMCert) == 0 {
		tlsConfig.InsecureSkipVerify = true
	} else {
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caPEMCert) {
			return nil, errors.New("Could not add RootCA pem")
		}
		tlsConfig.RootCAs = caPool
	}

	tr := defaultTransport()
	tr.TLSClientConfig = tlsConfig
	return &http.Client{Transport: tr}, nil
}

// defaultTransport returns a new http.Transport with similar default values to
// http.DefaultTransport, but with idle connections and keepalives disabled.
func defaultTransport() *http.Transport {
	transport := defaultPooledTransport()
	transport.DisableKeepAlives = true
	transport.MaxIdleConnsPerHost = -1
	return transport
}

// defaultPooledTransport returns a new http.Transport with similar default
// values to http.DefaultTransport.
func defaultPooledTransport() *http.Transport {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}
	return transport
}

// NewClient returns a new Docker client.
func (c *Config) NewClient() (*client.Client, error) {
	if c.Cert != "" || c.Key != "" {
		if c.Cert == "" || c.Key == "" {
			return nil, fmt.Errorf("cert_material, and key_material must be specified")
		}

		if c.CertPath != "" {
			return nil, fmt.Errorf("cert_path must not be specified")
		}

		httpClient, err := buildHTTPClientFromBytes([]byte(c.Ca), []byte(c.Cert), []byte(c.Key))
		if err != nil {
			return nil, err
		}

		// Note: don't change the order here, because the custom client
		// needs to be set first them we overwrite the other options: host, version
		return client.NewClientWithOpts(
			client.WithHTTPClient(httpClient),
			client.WithHost(c.Host),
			client.WithAPIVersionNegotiation(),
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
			client.WithAPIVersionNegotiation(),
		)
	}

	// If there is no cert information, then check for ssh://
	helper, err := connhelper.GetConnectionHelper(c.Host)
	if err != nil {
		return nil, err
	}
	if helper != nil {
		return client.NewClientWithOpts(
			client.WithHost(helper.Host),
			client.WithDialContext(helper.Dialer),
			client.WithAPIVersionNegotiation(),
		)
	}

	// If there is no ssh://, then just return the direct client
	return client.NewClientWithOpts(
		client.WithHost(c.Host),
		client.WithAPIVersionNegotiation(),
	)
}

// Data structure for holding data that we fetch from Docker.
type Data struct {
	DockerImages map[string]*types.ImageSummary
}

// ProviderConfig for the custom registry provider
type ProviderConfig struct {
	DefaultConfig *Config
	AuthConfigs   *AuthConfigs
	clientCache   map[Config]*client.Client
}

func (c *ProviderConfig) getConfig(d *schema.ResourceData) *Config {
	config := c.DefaultConfig

	if d != nil {
		resourceConfig := NewConfig(d)
		if resourceConfig.Host != "" {
			config = resourceConfig
		}
	}
	return config
}

func (c *ProviderConfig) MakeClient(ctx context.Context, d *schema.ResourceData) (*client.Client, error) {
	var dockerClient *client.Client
	var err error

	config := c.getConfig(d)
	dockerClient, found := c.clientCache[*config]
	if found {
		return dockerClient, nil
	}
	if config.Cert != "" || config.Key != "" {
		if config.Cert == "" || config.Key == "" {
			return nil, fmt.Errorf("cert_material, and key_material must be specified")
		}

		if config.CertPath != "" {
			return nil, fmt.Errorf("cert_path must not be specified")
		}

		httpClient, err := buildHTTPClientFromBytes([]byte(config.Ca), []byte(config.Cert), []byte(config.Key))
		if err != nil {
			return nil, err
		}

		// Note: don't change the order here, because the custom client
		// needs to be set first them we overwrite the other options: host, version
		dockerClient, err = client.NewClientWithOpts(
			client.WithHTTPClient(httpClient),
			client.WithHost(config.Host),
			client.WithAPIVersionNegotiation(),
		)
	} else if config.CertPath != "" {
		// If there is cert information, load it and use it.
		ca := filepath.Join(config.CertPath, "ca.pem")
		cert := filepath.Join(config.CertPath, "cert.pem")
		key := filepath.Join(config.CertPath, "key.pem")
		dockerClient, err = client.NewClientWithOpts(
			client.WithHost(config.Host),
			client.WithTLSClientConfig(ca, cert, key),
			client.WithAPIVersionNegotiation(),
		)
	} else if strings.HasPrefix(config.Host, "ssh://") {
		// If there is no cert information, then check for ssh://
		helper, err := connhelper.GetConnectionHelper(config.Host)
		if err != nil {
			return nil, err
		}
		if helper != nil {
			dockerClient, err = client.NewClientWithOpts(
				client.WithHost(helper.Host),
				client.WithDialContext(helper.Dialer),
				client.WithAPIVersionNegotiation(),
			)
		}
	} else {
		// If there is no ssh://, then just return the direct client
		dockerClient, err = client.NewClientWithOpts(
			client.WithHost(config.Host),
			client.WithAPIVersionNegotiation(),
		)
	}
	if err != nil {
		return nil, err
	}
	_, err = dockerClient.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("error pinging Docker server: %s", err)
	}
	c.clientCache[*config] = dockerClient

	return dockerClient, nil
}

// The registry address can be referenced in various places (registry auth, docker config file, image name)
// with or without the http(s):// prefix; this function is used to standardize the inputs
func normalizeRegistryAddress(address string) string {
	if !strings.HasPrefix(address, "https://") && !strings.HasPrefix(address, "http://") {
		return "https://" + address
	}
	return address
}
