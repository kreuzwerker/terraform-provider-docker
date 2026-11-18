package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// fakeDaemon answers every inspect call with the given status, as a daemon that
// refuses the request (500) or no longer has the object (404) would.
func fakeDaemon(t *testing.T, status int) *ProviderConfig {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"message":"` + http.StatusText(status) + `"}`))
	}))
	t.Cleanup(server.Close)
	return &ProviderConfig{
		DefaultConfig: &Config{
			Host:                     "tcp://" + strings.TrimPrefix(server.URL, "http://"),
			DisableDockerDaemonCheck: true,
		},
	}
}

func TestResourceRead_InspectError(t *testing.T) {
	reads := []struct {
		name     string
		resource *schema.Resource
		read     func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics
	}{
		{"config", resourceDockerConfig(), resourceDockerConfigRead},
		{"secret", resourceDockerSecret(), resourceDockerSecretRead},
		{"network", resourceDockerNetwork(), resourceDockerNetworkRead},
		{"plugin", resourceDockerPlugin(), resourceDockerPluginRead},
	}

	for _, tc := range reads {
		t.Run(tc.name+" keeps the resource when the daemon fails", func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, tc.resource.Schema, map[string]interface{}{})
			d.SetId("abc123")

			diags := tc.read(context.Background(), d, fakeDaemon(t, http.StatusInternalServerError))
			if !diags.HasError() {
				t.Fatalf("expected an error, got none")
			}
			if d.Id() != "abc123" {
				t.Fatalf("expected the resource to stay in state, id is now %q", d.Id())
			}
		})

		t.Run(tc.name+" removes the resource when it is gone", func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, tc.resource.Schema, map[string]interface{}{})
			d.SetId("abc123")

			diags := tc.read(context.Background(), d, fakeDaemon(t, http.StatusNotFound))
			if diags.HasError() {
				t.Fatalf("expected no error, got %v", diags)
			}
			if d.Id() != "" {
				t.Fatalf("expected the resource to be removed from state, id is %q", d.Id())
			}
		})
	}
}
