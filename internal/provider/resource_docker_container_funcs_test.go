package provider

import (
	"bytes"
	"context"
	"testing"

	"github.com/docker/docker/pkg/stdcopy"
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

func TestCopyContainerLogs_Demultiplex(t *testing.T) {
	var input bytes.Buffer
	stdoutWriter := stdcopy.NewStdWriter(&input, stdcopy.Stdout)
	stderrWriter := stdcopy.NewStdWriter(&input, stdcopy.Stderr)

	if _, err := stdoutWriter.Write([]byte("stdout\n")); err != nil {
		t.Fatalf("failed to write stdout log frame: %v", err)
	}
	if _, err := stderrWriter.Write([]byte("stderr\n")); err != nil {
		t.Fatalf("failed to write stderr log frame: %v", err)
	}

	var output bytes.Buffer
	if err := copyContainerLogs(&output, &input, false); err != nil {
		t.Fatalf("expected logs to be copied, got error: %v", err)
	}

	if got, want := output.String(), "stdout\nstderr\n"; got != want {
		t.Fatalf("unexpected logs output: got %q, want %q", got, want)
	}
}

func TestCopyContainerLogs_TTY(t *testing.T) {
	input := bytes.NewBufferString("tty-output\n")
	var output bytes.Buffer

	if err := copyContainerLogs(&output, input, true); err != nil {
		t.Fatalf("expected logs to be copied, got error: %v", err)
	}

	if got, want := output.String(), "tty-output\n"; got != want {
		t.Fatalf("unexpected logs output: got %q, want %q", got, want)
	}
}
