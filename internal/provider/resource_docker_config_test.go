package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerConfig_basic(t *testing.T) {
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
					name = "foo-config"
					data = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_config.foo", "name", "foo-config"),
					resource.TestCheckResourceAttr("docker_config.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
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
						ignore_changes = ["name"]
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
						ignore_changes = ["name"]
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

/////////////
// Helpers
/////////////
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
