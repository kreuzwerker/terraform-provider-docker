package provider

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
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
	SSHOpts  []string
	Ca       string
	Cert     string
	Key      string
	CertPath string
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
	helper, err := connhelper.GetConnectionHelperWithSSHOpts(c.Host, c.SSHOpts)
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
	DockerClient *client.Client
	AuthConfigs  *AuthConfigs
}

// The registry address can be referenced in various places (registry auth, docker config file, image name)
// with or without the http(s):// prefix; this function is used to standardize the inputs
// To support insecure (http) registries, if the address explicitly states "http://" we do not change it.
func normalizeRegistryAddress(address string) string {
	if strings.HasPrefix(address, "http://") {
		return address
	}
	if !strings.HasPrefix(address, "https://") && !strings.HasPrefix(address, "http://") {
		return "https://" + address
	}
	return address
}
