package actiontests

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestDockerContainerExportAction_exportsContainerFilesystem(t *testing.T) {
	preCheckDocker(t)

	containerName := fmt.Sprintf("tf-acc-docker-container-export-%d", time.Now().UnixNano())
	archivePath := filepath.Join(t.TempDir(), "container-export.tar")
	exportedFilePath := "tmp/container_export_action_file"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "docker_image" "busybox" {
  name = "busybox:1.35.0"
}

resource "docker_container" "target" {
  name     = %q
  image    = docker_image.busybox.image_id
  must_run = true
  command  = ["sh", "-c", "echo exported > /tmp/container_export_action_file && sleep 300"]

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.docker_container_export.export_container]
    }
  }
}

action "docker_container_export" "export_container" {
  config {
    container = docker_container.target.name
    output    = %q
  }
}
`, containerName, archivePath),
				PostApplyFunc: func() {
					stat, err := os.Stat(archivePath)
					if err != nil {
						t.Fatalf("expected export archive %q to exist: %s", archivePath, err)
					}
					if stat.Size() == 0 {
						t.Fatalf("expected export archive %q to be non-empty", archivePath)
					}

					found, err := tarContainsFile(archivePath, exportedFilePath)
					if err != nil {
						t.Fatalf("failed to inspect export archive: %s", err)
					}
					if !found {
						t.Fatalf("expected exported archive %q to contain %q", archivePath, exportedFilePath)
					}
				},
			},
		},
	})
}

func tarContainsFile(archivePath, targetPath string) (bool, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return false, err
	}
	defer file.Close() // nolint:errcheck

	tarReader := tar.NewReader(file)
	for {
		header, readErr := tarReader.Next()
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return false, readErr
		}

		if header.Name == targetPath {
			return true, nil
		}
	}

	return false, nil
}
