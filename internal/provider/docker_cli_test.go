package provider

import (
	"testing"

	"github.com/docker/docker/client"
)

func TestCreateAndInitDockerCliPreservesConfiguredSSHHost(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")

	apiClient, err := client.NewClientWithOpts(
		client.WithHost("http://docker.example.com"),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		t.Fatalf("failed to create API client: %s", err)
	}

	dockerCli, err := createAndInitDockerCli(apiClient, "ssh://user@example.com")
	if err != nil {
		t.Fatalf("failed to initialize Docker CLI: %s", err)
	}

	if dockerCli.DockerEndpoint().Host != "ssh://user@example.com" {
		t.Fatalf("expected docker CLI endpoint host %q, got %q", "ssh://user@example.com", dockerCli.DockerEndpoint().Host)
	}
}
