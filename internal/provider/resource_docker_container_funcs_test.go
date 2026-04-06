package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestDockerContainer_DeleteMissingContainer verifies that deleting a
// container that no longer exists on the Docker daemon succeeds.
func TestDockerContainer_DeleteMissingContainer(t *testing.T) {
	meta := &ProviderConfig{
		DefaultConfig: &Config{
			Host: "unix:///var/run/docker.sock",
		},
	}

	raw := map[string]interface{}{
		"name":                  "nonexistent",
		"image":                 "sha256:deadbeef",
		"attach":                false,
		"destroy_grace_seconds": 0,
		"remove_volumes":        true,
		"rm":                    false,
	}
	d := schema.TestResourceDataRaw(t, resourceDockerContainer().Schema, raw)
	d.SetId("nonexistent_container_id")

	diags := resourceDockerContainerDelete(context.Background(), d, meta)
	if diags.HasError() {
		t.Fatalf("expected no error deleting missing container, got: %v", diags)
	}
}
