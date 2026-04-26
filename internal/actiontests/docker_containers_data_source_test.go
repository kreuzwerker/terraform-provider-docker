package actiontests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestDockerContainersDataSource_roundtrip(t *testing.T) {
	preCheckDocker(t)

	containerName := fmt.Sprintf("tf-acc-docker-containers-%d", time.Now().UnixNano())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "docker_image" "busybox" {
  name = "busybox:1.35.0"
}

resource "docker_container" "target" {
  name     = %q
  image    = docker_image.busybox.image_id
  must_run = false
  command  = ["sh", "-c", "echo hello"]
}

data "docker_containers" "this" {
  depends_on = [docker_container.target]
}
`, containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.docker_containers.this", "id"),
					resource.TestCheckResourceAttrSet("data.docker_containers.this", "containers.#"),
					testCheckDockerContainersDataSourceContainsContainer("data.docker_containers.this", containerName),
				),
			},
		},
	})
}

func testCheckDockerContainersDataSourceContainsContainer(resourceName string, containerName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %q not found in state", resourceName)
		}

		for key, value := range rs.Primary.Attributes {
			if !strings.Contains(key, ".names.") {
				continue
			}

			if value == containerName || strings.TrimPrefix(value, "/") == containerName {
				return nil
			}
		}

		return fmt.Errorf("container %q not found in data source state: %#v", containerName, rs.Primary.Attributes)
	}
}
