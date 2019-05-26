package docker

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
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
					resource.TestCheckResourceAttr("data.docker_network.bridge", "driver", "bridge"),
					resource.TestCheckResourceAttr("data.docker_network.bridge", "internal", "false"),
					resource.TestCheckResourceAttr("data.docker_network.bridge", "scope", "local"),
				),
			},
		},
	})
}

const testAccDockerNetworkDataSourceConfig = `
data "docker_network" "bridge" {
	name = "bridge"
}
`
