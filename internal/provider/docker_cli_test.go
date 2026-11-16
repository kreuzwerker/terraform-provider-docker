package provider

import (
	"testing"

	"github.com/docker/docker/client"
)

func newTestAPIClient(t *testing.T, host string) *client.Client {
	t.Helper()
	apiClient, err := client.NewClientWithOpts(
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		t.Fatalf("failed to create API client: %s", err)
	}
	return apiClient
}

// buildx resolves the builder endpoint from the CLI's default context, so the
// CLI endpoint must be the configured remote host and not DOCKER_HOST or the
// local socket (#972).
func TestCreateAndInitDockerCliUsesConfiguredTCPHost(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")

	host := "tcp://docker.example.com:2375"
	dockerCli, err := createAndInitDockerCli(newTestAPIClient(t, host), host)
	if err != nil {
		t.Fatalf("failed to initialize Docker CLI: %s", err)
	}

	if got := dockerCli.DockerEndpoint().Host; got != host {
		t.Fatalf("expected docker CLI endpoint host %q, got %q", host, got)
	}
}

func TestCreateAndInitDockerCliPreservesConfiguredSSHHost(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")

	// For ssh:// hosts the API client is created with the connection
	// helper's dummy host, so the configured host must win.
	dockerCli, err := createAndInitDockerCli(newTestAPIClient(t, "http://docker.example.com"), "ssh://user@example.com")
	if err != nil {
		t.Fatalf("failed to initialize Docker CLI: %s", err)
	}

	if got := dockerCli.DockerEndpoint().Host; got != "ssh://user@example.com" {
		t.Fatalf("expected docker CLI endpoint host %q, got %q", "ssh://user@example.com", got)
	}
}

func TestCreateAndInitDockerCliFallsBackToDaemonHost(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")

	host := "tcp://docker.example.com:2376"
	dockerCli, err := createAndInitDockerCli(newTestAPIClient(t, host), "")
	if err != nil {
		t.Fatalf("failed to initialize Docker CLI: %s", err)
	}

	if got := dockerCli.DockerEndpoint().Host; got != host {
		t.Fatalf("expected docker CLI endpoint host %q, got %q", host, got)
	}
}
