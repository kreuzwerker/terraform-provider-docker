package provider

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDockerNetworkDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_network", "testAccDockerNetworkDataSourceBasic"),
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

func TestAccDockerNetworkDataSource_containers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_network", "testAccDockerNetworkDataSourceContainers"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_network.test", "containers.#", "1"),
					resource.TestCheckResourceAttr("data.docker_network.test", "containers.0.name", "tf-test-docker-network-data-source"),
					resource.TestCheckResourceAttr("data.docker_network.test", "containers.0.ipv4_address", "10.0.10.123/24"),
					resource.TestCheckResourceAttrSet("data.docker_network.test", "containers.0.container_id"),
					resource.TestCheckResourceAttrSet("data.docker_network.test", "containers.0.endpoint_id"),
					resource.TestCheckResourceAttrSet("data.docker_network.test", "containers.0.mac_address"),
				),
			},
		},
	})
}

func TestFlattenContainers(t *testing.T) {
	flattened := flattenContainers(map[string]network.EndpointResource{
		"bbb": {
			Name:        "second",
			EndpointID:  "endpoint-second",
			MacAddress:  "mac-second",
			IPv4Address: "10.0.10.3/24",
			IPv6Address: "fd00::3/64",
		},
		"aaa": {
			Name:        "first",
			EndpointID:  "endpoint-first",
			MacAddress:  "mac-first",
			IPv4Address: "10.0.10.2/24",
			IPv6Address: "fd00::2/64",
		},
	})

	expected := []networkContainerMap{
		{
			"container_id": "aaa",
			"name":         "first",
			"endpoint_id":  "endpoint-first",
			"mac_address":  "mac-first",
			"ipv4_address": "10.0.10.2/24",
			"ipv6_address": "fd00::2/64",
		},
		{
			"container_id": "bbb",
			"name":         "second",
			"endpoint_id":  "endpoint-second",
			"mac_address":  "mac-second",
			"ipv4_address": "10.0.10.3/24",
			"ipv6_address": "fd00::3/64",
		},
	}

	if !reflect.DeepEqual(flattened, expected) {
		t.Fatalf("unexpected flattened containers: got %#v, expected %#v", flattened, expected)
	}
}
