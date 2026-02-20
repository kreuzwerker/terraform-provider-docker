package provider

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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

func Test_complementTag(t *testing.T) {
	t.Parallel()
	data := []struct {
		title string
		image string
		exp   string
	}{
		{
			title: "alpine:3.11.6",
			image: "alpine:3.11.6",
			exp:   "alpine:3.11.6",
		},
		{
			title: "alpine",
			image: "alpine",
			exp:   "alpine:latest",
		},
	}
	for _, d := range data {
		d := d
		t.Run(d.title, func(t *testing.T) {
			t.Parallel()
			image := complementTag(d.image)
			if image != d.exp {
				t.Fatalf("want %v, got %v", d.exp, image)
			}
		})
	}
}

func Test_normalizePluginName(t *testing.T) {
	t.Parallel()
	data := []struct {
		title string
		image string
		isErr bool
		exp   string
	}{
		{
			title: "alpine:3.11.6",
			image: "alpine:3.11.6",
			exp:   "docker.io/library/alpine:3.11.6",
		},
		{
			title: "alpine",
			image: "alpine",
			exp:   "docker.io/library/alpine:latest",
		},
		{
			title: "vieux/sshfs",
			image: "vieux/sshfs",
			exp:   "docker.io/vieux/sshfs:latest",
		},
		{
			title: "docker.io/vieux/sshfs:latest",
			image: "docker.io/vieux/sshfs:latest",
			exp:   "docker.io/vieux/sshfs:latest",
		},
		{
			title: "docker.io/vieux/sshfs",
			image: "docker.io/vieux/sshfs",
			exp:   "docker.io/vieux/sshfs:latest",
		},
	}
	for _, d := range data {
		d := d
		t.Run(d.title, func(t *testing.T) {
			t.Parallel()
			image, err := normalizePluginName(d.image)
			if d.isErr {
				if err == nil {
					t.Fatal("error should be returned")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if image != d.exp {
				t.Fatalf("want %v, got %v", d.exp, image)
			}
		})
	}
}

func Test_getDockerPluginGrantPermissions(t *testing.T) {
	t.Parallel()
	data := []struct {
		title      string
		src        interface{}
		privileges types.PluginPrivileges
		exp        bool
		isErr      bool
	}{
		{
			title: "no privilege",
			src: schema.NewSet(dockerPluginGrantPermissionsSetFunc, []interface{}{
				map[string]interface{}{
					"name":  "network",
					"value": schema.NewSet(schema.HashString, []interface{}{"host"}),
				},
			}),
			exp: true,
		},
		{
			title: "basic",
			src: schema.NewSet(dockerPluginGrantPermissionsSetFunc, []interface{}{
				map[string]interface{}{
					"name":  "network",
					"value": schema.NewSet(schema.HashString, []interface{}{"host"}),
				},
			}),
			privileges: types.PluginPrivileges{
				{
					Name:  "network",
					Value: []string{"host"},
				},
			},
			exp: true,
		},
		{
			title: "permission denied 1",
			src: schema.NewSet(dockerPluginGrantPermissionsSetFunc, []interface{}{
				map[string]interface{}{
					"name": "network",
					"value": schema.NewSet(schema.HashString, []interface{}{
						"host",
					}),
				},
			}),
			privileges: types.PluginPrivileges{
				{
					Name:  "device",
					Value: []string{"/dev/fuse"},
				},
			},
			exp: false,
		},
		{
			title: "permission denied 2",
			src: schema.NewSet(dockerPluginGrantPermissionsSetFunc, []interface{}{
				map[string]interface{}{
					"name": "network",
					"value": schema.NewSet(schema.HashString, []interface{}{
						"host",
					}),
				},
				map[string]interface{}{
					"name": "mount",
					"value": schema.NewSet(schema.HashString, []interface{}{
						"/var/lib/docker/plugins/",
					}),
				},
			}),
			privileges: types.PluginPrivileges{
				{
					Name:  "network",
					Value: []string{"host"},
				},
				{
					Name:  "mount",
					Value: []string{"", "/var/lib/docker/plugins/"},
				},
			},
			exp: false,
		},
	}
	for _, d := range data {
		d := d
		t.Run(d.title, func(t *testing.T) {
			t.Parallel()

			f := getDockerPluginGrantPermissions(d.src)
			c := context.Background()

			b, err := f(c, d.privileges)
			if d.isErr {
				if err == nil {
					t.Fatal("error must be returned")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(d.exp, b) {
				t.Fatalf("want %v, got %v", d.exp, b)
			}
		})
	}
}

func TestAccDockerPlugin_basic(t *testing.T) {
	const resourceName = "docker_plugin.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				ResourceName: resourceName,
				Config:       loadTestConfiguration(t, RESOURCE, "docker_plugin", "testAccDockerPluginMinimum"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
			{
				ResourceName: resourceName,
				Config:       loadTestConfiguration(t, RESOURCE, "docker_plugin", "testAccDockerPluginAlias"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "sample:latest"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
			{
				ResourceName: resourceName,
				Config:       loadTestConfiguration(t, RESOURCE, "docker_plugin", "testAccDockerPluginDisableWhenSet"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "sample:latest"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "grant_all_permissions", "true"),
					resource.TestCheckResourceAttr(resourceName, "force_destroy", "true"),
					resource.TestCheckResourceAttr(resourceName, "enable_timeout", "60"),
				),
			},
			{
				ResourceName: resourceName,
				Config:       loadTestConfiguration(t, RESOURCE, "docker_plugin", "testAccDockerPluginDisabled"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "sample:latest"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "grant_all_permissions", "true"),
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

func TestAccDockerPlugin_full(t *testing.T) {
	const resourceName = "docker_plugin.test"
	var p types.Plugin

	testCheckPluginInspect := func(*terraform.State) error {
		if p.Enabled != false {
			return fmt.Errorf("Plugin Enabled is wrong: %v", p.Enabled)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				ResourceName: resourceName,
				Config:       loadTestConfiguration(t, RESOURCE, "docker_plugin", "testAccDockerPluginFull"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
					resource.TestCheckResourceAttr(resourceName, "alias", "sample:latest"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "force_destroy", "true"),
					resource.TestCheckResourceAttr(resourceName, "enable_timeout", "60"),
					resource.TestCheckResourceAttr(resourceName, "force_disable", "true"),
					resource.TestCheckResourceAttr(resourceName, "env.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "env.0", "DEBUG=1"),
					resource.TestCheckResourceAttr(resourceName, "grant_permissions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "grant_permissions.0.name", "network"),
					resource.TestCheckResourceAttr(resourceName, "grant_permissions.0.value.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "grant_permissions.0.value.0", "host"),
					testAccPluginCreated(resourceName, &p),
					testCheckPluginInspect,
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				ResourceName: resourceName,
				Config:       loadTestConfiguration(t, RESOURCE, "docker_plugin", "testAccDockerPluginGrantAllPermissions"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "docker.io/vieux/sshfs:latest"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				ResourceName: resourceName,
				Config:       loadTestConfiguration(t, RESOURCE, "docker_plugin", "testAccDockerPluginGrantPermissions"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "docker.io/vieux/sshfs:latest"),
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

func testAccPluginCreated(resourceName string, plugin *types.Plugin) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource with name '%s' not found in state", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := testAccProvider.Meta().(*ProviderConfig).MakeClient(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to create Docker client: %w", err)
		}
		inspectedPlugin, _, err := client.PluginInspectWithRaw(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Plugin with ID '%s': %w", rs.Primary.ID, err)
		}

		// we set the value to the pointer to be able to use the value
		// outside of the function
		plugin = inspectedPlugin
		return nil

	}
}
