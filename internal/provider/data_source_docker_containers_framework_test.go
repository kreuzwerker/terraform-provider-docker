package provider

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestDockerContainersDataSource_Metadata(t *testing.T) {
	dataSource := NewDockerContainersDataSource()
	resp := datasource.MetadataResponse{}

	dataSource.Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "docker",
	}, &resp)

	if resp.TypeName != "docker_containers" {
		t.Fatalf("expected type name docker_containers, got %s", resp.TypeName)
	}
}

func TestFlattenDockerContainers(t *testing.T) {
	ctx := context.Background()

	containers, diags := flattenDockerContainers(ctx, []container.Summary{
		{
			ID:      "abc123",
			Names:   []string{"/tf-test"},
			Image:   "busybox:latest",
			ImageID: "sha256:def456",
			Command: "sleep 300",
			Created: 42,
			State:   "running",
			Status:  "Up 1 second",
			Labels: map[string]string{
				"terraform": "true",
			},
		},
	})

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}

	if containers[0].ID.ValueString() != "abc123" {
		t.Fatalf("expected ID abc123, got %s", containers[0].ID.ValueString())
	}

	var names []string
	diags = containers[0].Names.ElementsAs(ctx, &names, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics decoding names: %v", diags)
	}
	if len(names) != 1 || names[0] != "/tf-test" {
		t.Fatalf("unexpected names: %#v", names)
	}

	var labels map[string]string
	diags = containers[0].Labels.ElementsAs(ctx, &labels, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics decoding labels: %v", diags)
	}
	if labels["terraform"] != "true" {
		t.Fatalf("expected terraform label to be true, got %#v", labels)
	}
}
