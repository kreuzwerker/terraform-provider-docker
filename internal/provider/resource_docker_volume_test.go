package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerVolume_basic(t *testing.T) {
	var v types.Volume

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
	var v types.Volume

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
	var v types.Volume

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

func checkDockerVolumeCreated(n string, volume *types.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		ctx := context.Background()
		client, err := testAccProvider.Meta().(*ProviderConfig).MakeClient(ctx, nil)
		if err != nil {
			return err
		}
		v, err := client.VolumeInspect(ctx, rs.Primary.ID)
		if err != nil {
			return err
		}

		*volume = v

		return nil
	}
}
