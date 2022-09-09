package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerConfig_basic(t *testing.T) {
	ctx := context.Background()
	var c swarm.Config

	testCheckConfigInspect := func(*terraform.State) error {
		if c.Spec.Name == "" {
			return errors.New("Config Spec.Name is empty")
		}

		if len(c.Spec.Data) == 0 {
			return errors.New("Config Spec.Data is empty")
		}

		if len(c.Spec.Labels) != 0 {
			return fmt.Errorf("Config Spec.Labels is wrong: %v", c.Spec.Labels)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testCheckDockerConfigDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_config" "foo" {
					name = "foo-config"
					data = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_config.foo", "name", "foo-config"),
					resource.TestCheckResourceAttr("docker_config.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
					testAccServiceConfigCreated("docker_config.foo", &c),
					testCheckConfigInspect,
				),
			},
			{
				ResourceName:      "docker_config.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDockerConfig_basicUpdatable(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testCheckDockerConfigDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_config" "foo" {
					name 			 = "tftest-myconfig-${replace(timestamp(),":", ".")}"
					data 			 = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="

					lifecycle {
						ignore_changes        = ["name"]
						create_before_destroy = true
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_config.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
				),
			},
			{
				Config: `
				resource "docker_config" "foo" {
					name 			 = "tftest-myconfig2-${replace(timestamp(),":", ".")}"
					data 			 = "U3VuIDI1IE1hciAyMDE4IDE0OjQ2OjE5IENFU1QK"

					lifecycle {
						ignore_changes        = ["name"]
						create_before_destroy = true
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_config.foo", "data", "U3VuIDI1IE1hciAyMDE4IDE0OjQ2OjE5IENFU1QK"),
				),
			},
			{
				ResourceName:      "docker_config.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// ///////////
// Helpers
// ///////////
func testCheckDockerConfigDestroy(ctx context.Context, s *terraform.State) error {
	client := testAccProvider.Meta().(*ProviderConfig).DockerClient
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "configs" {
			continue
		}

		id := rs.Primary.Attributes["id"]
		_, _, err := client.ConfigInspectWithRaw(ctx, id)

		if err == nil {
			return fmt.Errorf("Config with id '%s' still exists", id)
		}
		return nil
	}
	return nil
}

func testAccServiceConfigCreated(resourceName string, config *swarm.Config) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource with name '%s' not found in state", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		inspectedConfig, _, err := client.ConfigInspectWithRaw(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Config with ID '%s': %w", rs.Primary.ID, err)
		}

		// we set the value to the pointer to be able to use the value
		// outside of the function
		*config = inspectedConfig
		return nil

	}
}
