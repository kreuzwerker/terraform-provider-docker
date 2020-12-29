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
				Config:       testAccDockerPluginMinimum,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "disabled", "false"),
				),
			},
			{
				ResourceName: resourceName,
				Config:       testAccDockerPluginAlias,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "sample:latest"),
					resource.TestCheckResourceAttr(resourceName, "disabled", "false"),
				),
			},
			{
				ResourceName: resourceName,
				Config:       testAccDockerPluginDisableWhenSet,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "sample:latest"),
					resource.TestCheckResourceAttr(resourceName, "disabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "grant_all_permissions", "true"),
					resource.TestCheckResourceAttr(resourceName, "disable_when_set", "true"),
					resource.TestCheckResourceAttr(resourceName, "force_destroy", "true"),
					resource.TestCheckResourceAttr(resourceName, "enable_timeout", "60"),
				),
			},
			{
				ResourceName: resourceName,
				Config:       testAccDockerPluginDisabled,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "sample:latest"),
					resource.TestCheckResourceAttr(resourceName, "disabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "grant_all_permissions", "true"),
					resource.TestCheckResourceAttr(resourceName, "disable_when_set", "true"),
					resource.TestCheckResourceAttr(resourceName, "force_destroy", "true"),
					resource.TestCheckResourceAttr(resourceName, "enable_timeout", "60"),
					resource.TestCheckResourceAttr(resourceName, "force_disable", "true"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
			},
		},
	})
}

const testAccDockerPluginMinimum = `
resource "docker_plugin" "test" {
  plugin_reference = "docker.io/tiborvass/sample-volume-plugin:latest"
  force_destroy    = true
}`

const testAccDockerPluginAlias = `
resource "docker_plugin" "test" {
  plugin_reference = "docker.io/tiborvass/sample-volume-plugin:latest"
  alias            = "sample:latest"
  force_destroy    = true
}`

const testAccDockerPluginDisableWhenSet = `
resource "docker_plugin" "test" {
  plugin_reference              = "docker.io/tiborvass/sample-volume-plugin:latest"
  alias                         = "sample:latest"
  grant_all_permissions         = true
  disable_when_set              = true
  force_destroy                 = true
  enable_timeout                = 60
  args = [
    "DEBUG=1"
  ]
}`

const testAccDockerPluginDisabled = `
resource "docker_plugin" "test" {
  plugin_reference              = "docker.io/tiborvass/sample-volume-plugin:latest"
  alias                         = "sample:latest"
  disabled                      = true
  grant_all_permissions         = true
  disable_when_set              = true
  force_destroy                 = true
  force_disable                 = true
  enable_timeout                = 60
  args = [
    "DEBUG=1"
  ]
}`
