package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/registry"
)

// fakeRegistry answers every manifest request with the given status.
func fakeRegistry(t *testing.T, status int) (string, *ProviderConfig) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v2/app/manifests/") {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if status == http.StatusOK {
			w.Header().Set("Docker-Content-Digest", "sha256:0123")
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)

	host := strings.TrimPrefix(server.URL, "http://")
	return host, &ProviderConfig{
		AuthConfigs: &AuthConfigs{
			Configs: map[string]registry.AuthConfig{
				host: {ServerAddress: server.URL},
			},
		},
	}
}

func TestResourceDockerRegistryImageRead(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		wantError  bool
		wantID     string
		wantDigest string
	}{
		{name: "image present", status: http.StatusOK, wantID: "sha256:old", wantDigest: "sha256:0123"},
		{name: "image deleted from the registry", status: http.StatusNotFound, wantID: ""},
		{name: "registry error", status: http.StatusInternalServerError, wantError: true, wantID: "sha256:old"},
		{name: "registry unavailable", status: http.StatusServiceUnavailable, wantError: true, wantID: "sha256:old"},
		{name: "access denied", status: http.StatusForbidden, wantError: true, wantID: "sha256:old"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host, providerConfig := fakeRegistry(t, tc.status)
			d := resourceDockerRegistryImage().TestResourceData()
			if err := d.Set("name", host+"/app:1.0"); err != nil {
				t.Fatal(err)
			}
			d.SetId("sha256:old")

			diags := resourceDockerRegistryImageRead(context.Background(), d, providerConfig)
			if diags.HasError() != tc.wantError {
				t.Fatalf("HasError() = %v, want %v (%v)", diags.HasError(), tc.wantError, diags)
			}
			if d.Id() != tc.wantID {
				t.Fatalf("Id() = %q, want %q", d.Id(), tc.wantID)
			}
			if tc.wantDigest != "" && d.Get("sha256_digest").(string) != tc.wantDigest {
				t.Fatalf("sha256_digest = %q, want %q", d.Get("sha256_digest"), tc.wantDigest)
			}
		})
	}
}
