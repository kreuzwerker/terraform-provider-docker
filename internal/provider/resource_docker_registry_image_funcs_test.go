package provider

import "testing"

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
