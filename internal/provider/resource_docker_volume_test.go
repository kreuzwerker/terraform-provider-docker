package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/volume"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerVolume_basic(t *testing.T) {
	var v volume.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_volume" "foo" {
					name = "testAccDockerVolume_basic"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					checkDockerVolumeCreated("docker_volume.foo", &v),
					resource.TestCheckResourceAttr("docker_volume.foo", "id", "testAccDockerVolume_basic"),
					resource.TestCheckResourceAttr("docker_volume.foo", "name", "testAccDockerVolume_basic"),
				),
			},
			{
				ResourceName:      "docker_volume.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDockerVolume_full(t *testing.T) {
	var v volume.Volume

	testCheckVolumeInspect := func(*terraform.State) error {
		if v.Driver != "local" {
			return fmt.Errorf("Volume Driver is wrong: %v", v.Driver)
		}

		if v.Labels == nil ||
			!mapEquals("com.docker.compose.project", "test", v.Labels) ||
			!mapEquals("com.docker.compose.volume", "foo", v.Labels) {
			return fmt.Errorf("Volume Labels is wrong: %v", v.Labels)
		}

		if v.Options == nil ||
			!mapEquals("device", "/dev/sda2", v.Options) ||
			!mapEquals("type", "btrfs", v.Options) {
			return fmt.Errorf("Volume Options is wrong: %v", v.Options)
		}

		if v.Scope != "local" {
			return fmt.Errorf("Volume Scope is wrong: %v", v.Scope)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_volume", "testAccDockerVolumeFull"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_volume.foo", "id", "testAccDockerVolume_full"),
					resource.TestCheckResourceAttr("docker_volume.foo", "name", "testAccDockerVolume_full"),
					checkDockerVolumeCreated("docker_volume.foo", &v),
					testCheckVolumeInspect,
				),
			},
			{
				ResourceName:      "docker_volume.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDockerVolume_labels(t *testing.T) {
	var v volume.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_volume" "foo" {
					name = "test_foo"
					labels {
					  label = "com.docker.compose.project"
					  value = "test"
					}
					labels {
					  label = "com.docker.compose.volume"
					  value = "foo"
					}
				  }
				`,
				Check: resource.ComposeTestCheckFunc(
					checkDockerVolumeCreated("docker_volume.foo", &v),
					testCheckLabelMap("docker_volume.foo", "labels",
						map[string]string{
							"com.docker.compose.project": "test",
							"com.docker.compose.volume":  "foo",
						},
					),
				),
			},
			{
				ResourceName:      "docker_volume.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDockerVolume_RecreateAfterManualDelete(t *testing.T) {
	var v volume.Volume

	resourceName := "docker_volume.foo"
	volumeName := "testAccDockerVolume_manual_delete"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "docker_volume" "foo" {
						name = "%s"
					}
				`, volumeName),
				Check: resource.ComposeTestCheckFunc(
					checkDockerVolumeCreated(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "id", volumeName),
				),
			},
			{
				// Simulate manual deletion of the Docker volume
				PreConfig: func() {
					client := testAccProvider.Meta().(*ProviderConfig).DockerClient
					ctx := context.Background()
					err := client.VolumeRemove(ctx, volumeName, true)
					if err != nil {
						t.Fatalf("failed to manually remove docker volume: %v", err)
					}
				},
				Config: fmt.Sprintf(`
					resource "docker_volume" "foo" {
						name = "%s"
					}
				`, volumeName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func checkDockerVolumeCreated(n string, volumeToCheck *volume.Volume) resource.TestCheckFunc {
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

		*volumeToCheck = v

		return nil
	}
}
