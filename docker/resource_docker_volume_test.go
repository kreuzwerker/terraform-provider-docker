package docker

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestAccDockerVolume_basic(t *testing.T) {
	var v types.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerVolumeConfig,
				Check: resource.ComposeTestCheckFunc(
					checkDockerVolume("docker_volume.foo", &v),
					resource.TestCheckResourceAttr("docker_volume.foo", "id", "testAccDockerVolume_basic"),
					resource.TestCheckResourceAttr("docker_volume.foo", "name", "testAccDockerVolume_basic"),
				),
			},
		},
	})
}

func checkDockerVolume(n string, volume *types.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		ctx := context.Background()
		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		v, err := client.VolumeInspect(ctx, rs.Primary.ID)
		if err != nil {
			return err
		}

		*volume = v

		return nil
	}
}

const testAccDockerVolumeConfig = `
resource "docker_volume" "foo" {
	name = "testAccDockerVolume_basic"
}
`

func TestAccDockerVolume_labels(t *testing.T) {
	var v types.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerVolumeLabelsConfig,
				Check: resource.ComposeTestCheckFunc(
					checkDockerVolume("docker_volume.foo", &v),
					testAccVolumeLabel(&v, "com.docker.compose.project", "test"),
					testAccVolumeLabel(&v, "com.docker.compose.volume", "foo"),
				),
			},
		},
	})
}

func testAccVolumeLabel(volume *types.Volume, name string, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if volume.Labels[name] != value {
			return fmt.Errorf("Bad value for label '%s': %s", name, volume.Labels[name])
		}
		return nil
	}
}

const testAccDockerVolumeLabelsConfig = `
resource "docker_volume" "foo" {
  name = "test_foo"
  labels {
    "com.docker.compose.project" = "test"
    "com.docker.compose.volume"  = "foo"
  }
}
`
