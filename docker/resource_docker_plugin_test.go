package docker

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func Test_getDockerPluginEnv(t *testing.T) {
	t.Parallel()
	data := []struct {
		title string
		src   interface{}
		exp   []string
	}{
		{
			title: "nil",
		},
		{
			title: "basic",
			src:   schema.NewSet(schema.HashString, []interface{}{"DEBUG=1"}),
			exp:   []string{"DEBUG=1"},
		},
	}
	for _, d := range data {
		d := d
		t.Run(d.title, func(t *testing.T) {
			t.Parallel()
			envs := getDockerPluginEnv(d.src)
			if !reflect.DeepEqual(d.exp, envs) {
				t.Fatalf("want %v, got %v", d.exp, envs)
			}
		})
	}
}

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
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
			{
				ResourceName: resourceName,
				Config:       testAccDockerPluginAlias,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "sample:latest"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
			{
				ResourceName: resourceName,
				Config:       testAccDockerPluginDisableWhenSet,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "sample:latest"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
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
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
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

func TestAccDockerPlugin_grantAllPermissions(t *testing.T) {
	const resourceName = "docker_plugin.test"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				ResourceName: resourceName,
				Config:       testAccDockerPluginGrantAllPermissions,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/vieux/sshfs:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "vieux/sshfs:latest"),
					resource.TestCheckResourceAttr(resourceName, "grant_all_permissions", "true"),
				),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
			},
		},
	})
}

func TestAccDockerPlugin_grantPermissions(t *testing.T) {
	const resourceName = "docker_plugin.test"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				ResourceName: resourceName,
				Config:       testAccDockerPluginGrantPermissions,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/vieux/sshfs:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "vieux/sshfs:latest"),
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
  env = [
    "DEBUG=1"
  ]
}`

const testAccDockerPluginDisabled = `
resource "docker_plugin" "test" {
  plugin_reference              = "docker.io/tiborvass/sample-volume-plugin:latest"
  alias                         = "sample:latest"
  enabled                       = false
  grant_all_permissions         = true
  disable_when_set              = true
  force_destroy                 = true
  force_disable                 = true
  enable_timeout                = 60
  env = [
    "DEBUG=1"
  ]
}`

// To install this plugin, it is required to grant required permissions.
const testAccDockerPluginGrantAllPermissions = `
resource "docker_plugin" "test" {
  plugin_reference      = "docker.io/vieux/sshfs:latest"
  grant_all_permissions = true
  force_destroy         = true
}`

// To install this plugin, it is required to grant required permissions.
const testAccDockerPluginGrantPermissions = `
resource "docker_plugin" "test" {
  plugin_reference      = "docker.io/vieux/sshfs:latest"
  force_destroy         = true
  grant_permissions {
    name = "network"
    value = [
      "host"
    ]
  }
  grant_permissions {
    name = "mount"
    value = [
      "",
      "/var/lib/docker/plugins/"
    ]
  }
  grant_permissions {
    name = "device"
    value = [
      "/dev/fuse"
    ]
  }
  grant_permissions {
    name = "capabilities"
    value = [
      "CAP_SYS_ADMIN"
    ]
  }
}`
