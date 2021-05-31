package provider

import (
	"os/exec"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDockerPluginDataSource_basic(t *testing.T) {
	pluginName := "tiborvass/sample-volume-plugin"
	// This fails if the plugin is already installed.
	if err := exec.Command("docker", "plugin", "install", pluginName).Run(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := exec.Command("docker", "plugin", "rm", "-f", pluginName).Run(); err != nil {
			t.Logf("failed to remove the Docker plugin %s: %v", pluginName, err)
		}
	}()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_plugin", "testAccDockerPluginDataSourceBasic"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_plugin.test", "plugin_reference", "docker.io/tiborvass/sample-volume-plugin:latest"),
				),
			},
		},
	})
}
