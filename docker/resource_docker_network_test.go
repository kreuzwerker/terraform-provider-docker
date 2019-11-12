package docker

import (
	"fmt"
	"testing"

	"context"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccDockerNetwork_basic(t *testing.T) {
	var n types.NetworkResource

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerNetworkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork("docker_network.foo", &n),
				),
			},
		},
	})
}

func testAccNetwork(n string, network *types.NetworkResource) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		networks, err := client.NetworkList(context.Background(), types.NetworkListOptions{})
		if err != nil {
			return err
		}

		for _, n := range networks {
			if n.ID == rs.Primary.ID {
				inspected, err := client.NetworkInspect(context.Background(), n.ID, types.NetworkInspectOptions{})
				if err != nil {
					return fmt.Errorf("Network could not be obtained: %s", err)
				}
				*network = inspected
				return nil
			}
		}

		return fmt.Errorf("Network not found: %s", rs.Primary.ID)
	}
}

const testAccDockerNetworkConfig = `
resource "docker_network" "foo" {
  name = "bar"
}
`

func TestAccDockerNetwork_internal(t *testing.T) {
	var n types.NetworkResource

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerNetworkInternalConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork("docker_network.foo", &n),
					testAccNetworkInternal(&n, true),
				),
			},
		},
	})
}

func testAccNetworkInternal(network *types.NetworkResource, internal bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if network.Internal != internal {
			return fmt.Errorf("Bad value for attribute 'internal': %t", network.Internal)
		}
		return nil
	}
}

const testAccDockerNetworkInternalConfig = `
resource "docker_network" "foo" {
  name = "bar"
  internal = true
}
`

func TestAccDockerNetwork_attachable(t *testing.T) {
	var n types.NetworkResource

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerNetworkAttachableConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork("docker_network.foo", &n),
					testAccNetworkAttachable(&n, true),
				),
			},
		},
	})
}

func testAccNetworkAttachable(network *types.NetworkResource, attachable bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if network.Attachable != attachable {
			return fmt.Errorf("Bad value for attribute 'attachable': %t", network.Attachable)
		}
		return nil
	}
}

const testAccDockerNetworkAttachableConfig = `
resource "docker_network" "foo" {
  name = "bar"
  attachable = true
}
`

//func TestAccDockerNetwork_ingress(t *testing.T) {
//	var n types.NetworkResource
//
//	resource.Test(t, resource.TestCase{
//		PreCheck:  func() { testAccPreCheck(t) },
//		Providers: testAccProviders,
//		Steps: []resource.TestStep{
//			resource.TestStep{
//				Config: testAccDockerNetworkIngressConfig,
//				Check: resource.ComposeTestCheckFunc(
//					testAccNetwork("docker_network.foo", &n),
//					testAccNetworkIngress(&n, true),
//				),
//			},
//		},
//	})
//}
//
//func testAccNetworkIngress(network *types.NetworkResource, internal bool) resource.TestCheckFunc {
//	return func(s *terraform.State) error {
//		if network.Internal != internal {
//			return fmt.Errorf("Bad value for attribute 'ingress': %t", network.Ingress)
//		}
//		return nil
//	}
//}
//
//const testAccDockerNetworkIngressConfig = `
//resource "docker_network" "foo" {
//  name = "bar"
//  ingress = true
//}
//`

func TestAccDockerNetwork_ipv4(t *testing.T) {
	var n types.NetworkResource

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerNetworkIPv4Config,
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork("docker_network.foo", &n),
					testAccNetworkIPv4(&n, true),
				),
			},
		},
	})
}

func testAccNetworkIPv4(network *types.NetworkResource, internal bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(network.IPAM.Config) != 1 {
			return fmt.Errorf("Bad value for IPAM configuration count: %d", len(network.IPAM.Config))
		}
		if network.IPAM.Config[0].Subnet != "10.0.1.0/24" {
			return fmt.Errorf("Bad value for attribute 'subnet': %v", network.IPAM.Config[0].Subnet)
		}
		return nil
	}
}

const testAccDockerNetworkIPv4Config = `
resource "docker_network" "foo" {
  name = "bar"
  ipam_config {
    subnet = "10.0.1.0/24"
  }
}
`

func TestAccDockerNetwork_ipv6(t *testing.T) {
	var n types.NetworkResource

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerNetworkIPv6Config,
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork("docker_network.foo", &n),
					testAccNetworkIPv6(&n, true),
				),
			},
		},
	})
}

func testAccNetworkIPv6(network *types.NetworkResource, internal bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !network.EnableIPv6 {
			return fmt.Errorf("Bad value for attribute 'ipv6': %t", network.EnableIPv6)
		}
		if len(network.IPAM.Config) != 2 {
			return fmt.Errorf("Bad value for IPAM configuration count: %d", len(network.IPAM.Config))
		}
		if network.IPAM.Config[1].Subnet != "fd00::1/64" {
			return fmt.Errorf("Bad value for attribute 'subnet': %v", network.IPAM.Config[0].Subnet)
		}
		return nil
	}
}

const testAccDockerNetworkIPv6Config = `
resource "docker_network" "foo" {
  name = "bar"
  ipv6 = true
  ipam_config {
    subnet = "fd00::1/64"
  }
}
`

func TestAccDockerNetwork_labels(t *testing.T) {
	var n types.NetworkResource

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerNetworkLabelsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork("docker_network.foo", &n),
					testCheckLabelMap("docker_network.foo", "labels",
						map[string]string{
							"com.docker.compose.network": "foo",
							"com.docker.compose.project": "test",
						},
					),
				),
			},
		},
	})
}

func testAccNetworkLabel(network *types.NetworkResource, name string, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if network.Labels[name] != value {
			return fmt.Errorf("Bad value for label '%s': %s", name, network.Labels[name])
		}
		return nil
	}
}

const testAccDockerNetworkLabelsConfig = `
resource "docker_network" "foo" {
  name = "test_foo"
  labels {
    label = "com.docker.compose.network"
    value = "foo"
  }
  labels {
    label = "com.docker.compose.project"
    value = "test"
  }
}
`
