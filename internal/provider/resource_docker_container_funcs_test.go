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

func TestResourceDataHasNonNullConfigAttribute(t *testing.T) {
	testCases := []struct {
		name      string
		rawConfig map[string]interface{}
		attribute string
		want      bool
	}{
		{
			name:      "attribute absent",
			rawConfig: map[string]interface{}{},
			attribute: "devices",
			want:      false,
		},
		{
			name: "devices present as empty set",
			rawConfig: map[string]interface{}{
				"devices": []interface{}{},
			},
			attribute: "devices",
			want:      true,
		},
		{
			name: "device_requests present as empty set",
			rawConfig: map[string]interface{}{
				"device_requests": []interface{}{},
			},
			attribute: "device_requests",
			want:      true,
		},
		{
			name: "gpus present as empty string",
			rawConfig: map[string]interface{}{
				"gpus": "",
			},
			attribute: "gpus",
			want:      true,
		},
		{
			name: "unknown attribute",
			rawConfig: map[string]interface{}{
				"name": "foo",
			},
			attribute: "non_existing_attribute",
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, resourceDockerContainer().Schema, tc.rawConfig)
			got := resourceDataHasNonNullConfigAttribute(d, tc.attribute)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
