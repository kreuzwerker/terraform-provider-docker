package docker

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDockerNetworkDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerNetworkDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_network.bridge", "name", "bridge"),
					testAccDockerNetworkDataSourceIPAMRead,
					resource.TestCheckResourceAttr("data.docker_network.bridge", "driver", "bridge"),
					resource.TestCheckResourceAttr("data.docker_network.bridge", "internal", "false"),
					resource.TestCheckResourceAttr("data.docker_network.bridge", "scope", "local"),
				),
			},
		},
	})
}

func testAccDockerNetworkDataSourceIPAMRead(state *terraform.State) error {
	bridge := state.RootModule().Resources["data.docker_network.bridge"]
	if bridge == nil {
		return fmt.Errorf("unable to find data.docker_network.bridge")
	}
	attr := bridge.Primary.Attributes["ipam_config.#"]
	numberOfReadConfig, err := strconv.Atoi(attr)
	if err != nil {
		return err
	}
	if numberOfReadConfig < 1 {
		return fmt.Errorf("unable to find any ipam_config")
	}
	return nil
}

const testAccDockerNetworkDataSourceConfig = `
data "docker_network" "bridge" {
	name = "bridge"
}
`
