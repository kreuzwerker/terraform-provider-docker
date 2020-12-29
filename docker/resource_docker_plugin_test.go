package docker

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDockerPlugin_basic(t *testing.T) {
	const resourceName = "docker_plugin.test"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				ResourceName: resourceName,
				Config:       testAccDockerPluginConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
			},
		},
	})
}

const testAccDockerPluginConfig = `
resource "docker_plugin" "test" {
  plugin_reference = "docker.io/tiborvass/sample-volume-plugin:latest"
	force_destroy    = true
}`
