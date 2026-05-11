package provider

import (
	"testing"

	"github.com/docker/docker/api/types/registry"
)

func TestBuildAuthConfigFromResource_OptionalCredentials(t *testing.T) {
	authConfig := buildAuthConfigFromResource([]interface{}{
		map[string]interface{}{
			"address": "europe-west4-docker.pkg.dev",
		},
	})

	if authConfig.ServerAddress != "https://europe-west4-docker.pkg.dev" {
		t.Fatalf("want normalized address https://europe-west4-docker.pkg.dev, got %s", authConfig.ServerAddress)
	}

	if authConfig.Username != "" {
		t.Fatalf("want empty username, got %s", authConfig.Username)
	}

	if authConfig.Password != "" {
		t.Fatalf("want empty password, got %s", authConfig.Password)
	}
}

func TestBuildAuthConfigFromResource_WithCredentials(t *testing.T) {
	authConfig := buildAuthConfigFromResource([]interface{}{
		map[string]interface{}{
			"address":  "europe-west4-docker.pkg.dev",
			"username": "test-user",
			"password": "test-password",
		},
	})

	if authConfig.ServerAddress != "https://europe-west4-docker.pkg.dev" {
		t.Fatalf("want normalized address https://europe-west4-docker.pkg.dev, got %s", authConfig.ServerAddress)
	}

	if authConfig.Username != "test-user" {
		t.Fatalf("want username test-user, got %s", authConfig.Username)
	}

	if authConfig.Password != "test-password" {
		t.Fatalf("want password test-password, got %s", authConfig.Password)
	}
}

func TestGetAuthConfigForRegistry_DockerHubAliasLookup(t *testing.T) {
	providerConfig := &ProviderConfig{
		AuthConfigs: &AuthConfigs{
			Configs: map[string]registry.AuthConfig{
				"index.docker.io": {
					Username:      "docker-user",
					Password:      "docker-pass",
					ServerAddress: "https://index.docker.io/v1/",
				},
			},
		},
	}

	authConfig, err := getAuthConfigForRegistry("registry-1.docker.io", providerConfig)
	if err != nil {
		t.Fatalf("unexpected getAuthConfigForRegistry error: %s", err)
	}

	if authConfig.Username != "docker-user" {
		t.Fatalf("want username docker-user, got %s", authConfig.Username)
	}

	if authConfig.Password != "docker-pass" {
		t.Fatalf("want password docker-pass, got %s", authConfig.Password)
	}
}
