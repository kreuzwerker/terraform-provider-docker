package actiontests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestDockerImageSaveAction_savesImageToArchive(t *testing.T) {
	preCheckDocker(t)

	archivePath := filepath.Join(t.TempDir(), "saved-image.tar")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "docker_image" "busybox" {
  name         = "busybox:1.35.0"
  keep_locally = true
}

resource "terraform_data" "trigger" {
  depends_on = [docker_image.busybox]

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.docker_image_save.save_busybox]
    }
  }
}

action "docker_image_save" "save_busybox" {
  config {
    images   = [docker_image.busybox.name]
    output   = %q
    platform = "linux/amd64"
  }
}
`, archivePath),
				PostApplyFunc: func() {
					stat, err := os.Stat(archivePath)
					if err != nil {
						t.Fatalf("expected image archive %q to exist: %s", archivePath, err)
					}

					if stat.Size() == 0 {
						t.Fatalf("expected image archive %q to be non-empty", archivePath)
					}
				},
			},
		},
	})
}

func runDockerCommand(args ...string) error {
	command := exec.Command("docker", args...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}

	return nil
}

func TestDockerImageLoadAction_loadsImageFromArchive(t *testing.T) {
	preCheckDocker(t)

	imageTag := fmt.Sprintf("tf-acc-docker-image-load-%d:latest", time.Now().UnixNano())
	archivePath := filepath.Join(t.TempDir(), "load-image.tar")

	defer func() {
		_ = runDockerCommand("image", "rm", "-f", imageTag)
	}()

	if err := runDockerCommand("pull", "busybox:1.35.0"); err != nil {
		t.Fatalf("failed to pull busybox: %s", err)
	}
	if err := runDockerCommand("tag", "busybox:1.35.0", imageTag); err != nil {
		t.Fatalf("failed to tag image: %s", err)
	}
	if err := runDockerCommand("save", "-o", archivePath, imageTag); err != nil {
		t.Fatalf("failed to save image archive: %s", err)
	}
	if err := runDockerCommand("image", "rm", "-f", imageTag); err != nil {
		t.Fatalf("failed to remove tagged image before load test: %s", err)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.docker_image_load.load_archive]
    }
  }
}

action "docker_image_load" "load_archive" {
  config {
    source   = %q
    quiet    = true
    platform = "linux/amd64"
  }
}
`, archivePath),
				PostApplyFunc: func() {
					if err := runDockerCommand("image", "inspect", imageTag); err != nil {
						t.Fatalf("expected loaded image %q to exist: %s", imageTag, err)
					}
				},
			},
		},
	})
}
