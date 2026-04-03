package provider

import (
	"strings"
	"testing"
)

func TestGetAuthConfigFromConfigFile_PrefersCanonicalDockerHubEntry(t *testing.T) {
	content := `{
		"auths": {
			"https://index.docker.io/v1/": {
				"auth": "dXNlcjpkY2tyX3BhdF9hYmM="
			},
			"https://index.docker.io/v1/access-token": {
				"auth": "dXNlcjpleUpoYkdjaU9pSldU"
			},
			"https://index.docker.io/v1/refresh-token": {
				"auth": "dXNlcjp2MS5NV1Z1eDI="
			}
		}
	}`

	cfg, err := loadConfigFile(strings.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected loadConfigFile error: %s", err)
	}

	auth, err := getAuthConfigFromConfigFile(cfg, "index.docker.io")
	if err != nil {
		t.Fatalf("unexpected getAuthConfigFromConfigFile error: %s", err)
	}

	if auth.Username != "user" {
		t.Fatalf("want username user, got %s", auth.Username)
	}

	if auth.Password != "dckr_pat_abc" {
		t.Fatalf("want canonical docker hub password dckr_pat_abc, got %s", auth.Password)
	}
}
