package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Config is the structure that stores the configuration to talk to a
// Docker API compatible host.
type Config struct {
	Host                     string
	SSHOpts                  []string
	Ca                       string
	Cert                     string
	Key                      string
	CertPath                 string
	DisableDockerDaemonCheck bool
}

func (c *Config) Hash() uint64 {
	var SSHOpts []string

	copy(SSHOpts, c.SSHOpts)
	sort.Strings(SSHOpts)

	hash := fnv.New64()
	_, err := hash.Write([]byte(strings.Join([]string{
		c.Host,
		c.Ca,
		c.Cert,
		c.Key,
		c.CertPath,
		strings.Join(SSHOpts, "|")},
		"|",
	)))

	if err != nil {
		panic(err)
	}

	return hash.Sum64()
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
			return nil, errors.New("could not add RootCA pem")
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
func (c *ProviderConfig) MakeClient(ctx context.Context, d *schema.ResourceData) (*client.Client, error) {
	var dockerClient *client.Client
	var err error

	config := *c.DefaultConfig
	configHash := config.Hash()
	cached, found := c.clientCache.Load(configHash)

	if found {
		log.Printf("[DEBUG] Found cached client! Hash:%d Host:%s", configHash, config.Host)
		return cached.(*client.Client), nil
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
		if err != nil {
			return nil, err
		}
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
		if err != nil {
			return nil, err
		}
	} else if strings.HasPrefix(config.Host, "ssh://") {
		// If there is no cert information, then check for ssh://
		helper, err := connhelper.GetConnectionHelperWithSSHOpts(config.Host, config.SSHOpts)
		if err != nil {
			return nil, err
		}
		if helper != nil {
			dockerClient, err = client.NewClientWithOpts(
				client.WithHost(helper.Host),
				client.WithDialContext(helper.Dialer),
				client.WithAPIVersionNegotiation(),
			)
			if err != nil {
				return nil, err
			}
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

	if config.DisableDockerDaemonCheck {
		log.Printf("[DEBUG] Skipping Docker daemon check")
	} else {
		_, err = dockerClient.Ping(ctx)
		if err != nil {
			return nil, fmt.Errorf("Error pinging Docker server, please make sure that %s is reachable and has a  '_ping' endpoint. Error: %s", config.Host, err)
		}
		_, err = dockerClient.ServerVersion(ctx)
		if err != nil {
			log.Printf("[WARN] Error connecting to Docker daemon. Is your endpoint a valid docker host? This warning will be changed to an error in the next major version. Error: %s", err)
		}
	}

	c.clientCache.LoadOrStore(configHash, dockerClient)
	log.Printf("[INFO] New client with Hash:%d Host:%s", configHash, config.Host)

	return dockerClient, nil
}

// Data structure for holding data that we fetch from Docker.
type Data struct {
	DockerImages map[string]*image.Summary
}

// ProviderConfig for the custom registry provider
type ProviderConfig struct {
	DefaultConfig *Config
	Hosts         map[string]*schema.ResourceData
	AuthConfigs   *AuthConfigs
	clientCache   sync.Map
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
