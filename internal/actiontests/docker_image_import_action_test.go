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

func TestDockerImageImportAction_importsTarballIntoImage(t *testing.T) {
	preCheckDocker(t)

	containerName := fmt.Sprintf("tf-acc-docker-import-%d", time.Now().UnixNano())
	imageRef := fmt.Sprintf("tf-acc-docker-imported-%d:latest", time.Now().UnixNano())
	defer func() {
		_ = exec.Command("docker", "image", "rm", "-f", imageRef).Run()
	}()

	tempDir := t.TempDir()
	tarPath := filepath.Join(tempDir, "import.tar")

	createCmd := exec.Command("docker", "run", "--name", containerName, "-d", "busybox:1.35.0", "sh", "-c", "sleep 300")
	if output, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create container for export: %s: %s", err, string(output))
	}
	defer func() {
		_ = exec.Command("docker", "rm", "-f", containerName).Run()
	}()

	createFileCmd := exec.Command("docker", "exec", containerName, "sh", "-c", "echo imported > /tmp/docker_import_action_file")
	if output, err := createFileCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create file inside export container: %s: %s", err, string(output))
	}

	exportFile, err := os.Create(tarPath)
	if err != nil {
		t.Fatalf("failed to create tar file: %s", err)
	}

	exportCmd := exec.Command("docker", "export", containerName)
	exportCmd.Stdout = exportFile
	if err := exportCmd.Run(); err != nil {
		_ = exportFile.Close()
		t.Fatalf("failed to export container: %s", err)
	}
	if err := exportFile.Close(); err != nil {
		t.Fatalf("failed to close tar file: %s", err)
	}

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

resource "docker_container" "trigger" {
  name     = %q
  image    = docker_image.busybox.image_id
  must_run = true
  command  = ["sh", "-c", "sleep 300"]

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.docker_image_import.import_export]
    }
  }
}

action "docker_image_import" "import_export" {
  config {
    source    = %q
    reference = %q
    message   = "imported from docker export"
    changes   = ["CMD [\"sh\"]"]
    platform  = "linux/amd64"
  }
}
`, containerName, tarPath, imageRef),

				PostApplyFunc: func() {
					checkCmd := exec.Command("docker", "image", "inspect", imageRef)
					if output, err := checkCmd.CombinedOutput(); err != nil {
						t.Fatalf("expected imported image %q to exist: %s: %s", imageRef, err, string(output))
					}

					runCmd := exec.Command("docker", "run", "--rm", imageRef, "sh", "-c", "test -f /tmp/docker_import_action_file")
					if output, err := runCmd.CombinedOutput(); err != nil {
						t.Fatalf("expected imported image %q to contain the exported file: %s: %s", imageRef, err, string(output))
					}
				},
			},
		},
	})
}
