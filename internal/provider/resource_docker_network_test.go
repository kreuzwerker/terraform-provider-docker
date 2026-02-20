package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerNetwork_basic(t *testing.T) {
	var n network.Inspect
	resourceName := "docker_network.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_network", "testAccDockerNetworkConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork(resourceName, &n),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDockerNetwork_full(t *testing.T) {
	var n network.Inspect
	resourceName := "docker_network.foo"

	testCheckNetworkInspect := func(*terraform.State) error {
		if n.Scope == "" || n.Scope != "local" {
			return fmt.Errorf("Network Scope is wrong: %v", n.Scope)
		}

		if n.Driver == "" || n.Driver != "bridge" {
			return fmt.Errorf("Network Driver is wrong: %v", n.Driver)
		}

		if n.EnableIPv6 != false {
			return fmt.Errorf("Network EnableIPv6 is wrong: %v", n.EnableIPv6)
		}

		if n.IPAM.Driver == "" ||
			n.IPAM.Options != nil ||
			len(n.IPAM.Config) != 1 ||
			n.IPAM.Config[0].Gateway != "" ||
			n.IPAM.Config[0].IPRange != "" ||
			n.IPAM.Config[0].AuxAddress != nil ||
			n.IPAM.Config[0].Subnet != "10.0.1.0/24" ||
			n.IPAM.Driver != "default" {
			return fmt.Errorf("Network IPAM is wrong: %v", n.IPAM)
		}

		if n.Internal != true {
			return fmt.Errorf("Network Internal is wrong: %v", n.Internal)
		}

		if n.Attachable != false {
			return fmt.Errorf("Network Attachable is wrong: %v", n.Attachable)
		}

		if n.Ingress != false {
			return fmt.Errorf("Network Ingress is wrong: %v", n.Ingress)
		}

		if n.ConfigFrom.Network != "" {
			return fmt.Errorf("Network ConfigFrom is wrong: %v", n.ConfigFrom)
		}

		if n.ConfigOnly != false {
			return fmt.Errorf("Network ConfigOnly is wrong: %v", n.ConfigOnly)
		}

		if n.Containers == nil || len(n.Containers) != 0 {
			return fmt.Errorf("Network Containers is wrong: %v", n.Containers)
		}

		if n.Options == nil || len(n.Options) != 0 {
			return fmt.Errorf("Network Options is wrong: %v", n.Options)
		}

		if n.Labels == nil ||
			len(n.Labels) != 2 ||
			!mapEquals("com.docker.compose.network", "foo", n.Labels) ||
			!mapEquals("com.docker.compose.project", "test", n.Labels) {
			return fmt.Errorf("Network Labels is wrong: %v", n.Labels)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_network", "testAccDockerNetworkConfigFull"),
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork(resourceName, &n),
					testCheckNetworkInspect,
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TODO mavogel: add full network config test in #74 (import resources)

func testAccNetwork(n string, networkToCheck *network.Inspect) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := testAccProvider.Meta().(*ProviderConfig).MakeClient(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to create Docker client: %w", err)
		}
		networks, err := client.NetworkList(ctx, network.ListOptions{})
		if err != nil {
			return err
		}

		for _, n := range networks {
			if n.ID == rs.Primary.ID {
				inspected, err := client.NetworkInspect(ctx, n.ID, network.InspectOptions{})
				if err != nil {
					return fmt.Errorf("Network could not be obtained: %s", err)
				}
				*networkToCheck = inspected
				return nil
			}
		}

		return fmt.Errorf("Network not found: %s", rs.Primary.ID)
	}
}

func TestAccDockerNetwork_internal(t *testing.T) {
	var n network.Inspect
	resourceName := "docker_network.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_network", "testAccDockerNetworkInternalConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork(resourceName, &n),
					testAccNetworkInternal(&n, true),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccNetworkInternal(network *network.Inspect, internal bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if network.Internal != internal {
			return fmt.Errorf("Bad value for attribute 'internal': %t", network.Internal)
		}
		return nil
	}
}

func TestAccDockerNetwork_attachable(t *testing.T) {
	var n network.Inspect
	resourceName := "docker_network.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_network", "testAccDockerNetworkAttachableConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork(resourceName, &n),
					testAccNetworkAttachable(&n, true),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccNetworkAttachable(network *network.Inspect, attachable bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if network.Attachable != attachable {
			return fmt.Errorf("Bad value for attribute 'attachable': %t", network.Attachable)
		}
		return nil
	}
}

func TestAccDockerNetwork_ingress(t *testing.T) {
	ctx := context.Background()
	var n network.Inspect

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			// as we join the swarm an ingress network is created by default
			// As only one can exist, we remove it for the test
			removeSwarmIngressNetwork(ctx, t)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_network", "testAccDockerNetworkIngressConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork("docker_network.foo", &n),
					testAccNetworkIngress(&n, true),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			// we leave the swarm because in the next testAccPreCheck
			// the node will join the swarm again
			// and so recreate the default swarm ingress network
			return nodeLeaveSwarm(ctx, t)
		},
	})
}

func removeSwarmIngressNetwork(ctx context.Context, t *testing.T) {
	client, err := testAccProvider.Meta().(*ProviderConfig).MakeClient(ctx, nil)
	if err != nil {
		t.Errorf("failed to create Docker client: %v", err)
	}
	networks, err := client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		t.Errorf("failed to list swarm networks: %v", err)
	}
	var ingressNetworkID string
	for _, network := range networks {
		if network.Ingress {
			ingressNetworkID = network.ID
			break
		}
	}
	err = client.NetworkRemove(ctx, ingressNetworkID)
	if err != nil {
		t.Errorf("failed to remove swarm ingress network '%s': %v", ingressNetworkID, err)
	}
}

func nodeLeaveSwarm(ctx context.Context, t *testing.T) error {
	client, err := testAccProvider.Meta().(*ProviderConfig).MakeClient(ctx, nil)
	if err != nil {
		t.Errorf("failed to create Docker client: %v", err)
	}
	force := true
	err = client.SwarmLeave(ctx, force)
	if err != nil {
		t.Errorf("node failed to leave the swarm: %v", err)
	}
	return nil
}

func testAccNetworkIngress(network *network.Inspect, ingress bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if network.Ingress != ingress {
			return fmt.Errorf("Bad value for attribute 'ingress': %t", network.Ingress)
		}
		return nil
	}
}

func TestAccDockerNetwork_ipv4(t *testing.T) {
	var n network.Inspect
	resourceName := "docker_network.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_network", "testAccDockerNetworkIPv4Config"),
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork(resourceName, &n),
					testAccNetworkIPv4(&n, true),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccNetworkIPv4(network *network.Inspect, internal bool) resource.TestCheckFunc {
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

func TestAccDockerNetwork_ipv6(t *testing.T) {
	t.Skip("TODO mavogel: need to fix ipv6 network state")
	var n network.Inspect
	resourceName := "docker_network.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_network", "testAccDockerNetworkIPv6Config"),
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork(resourceName, &n),
					testAccNetworkIPv6(&n, true),
				),
			},
			// TODO mavogel: ipam config goes from 2->1
			// probably suppress diff -> #74 (import resources)
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccNetworkIPv6(network *network.Inspect, internal bool) resource.TestCheckFunc { //nolint:unused
	return func(s *terraform.State) error {
		if !network.EnableIPv6 {
			return fmt.Errorf("Bad value for attribute 'ipv6': %t", network.EnableIPv6)
		}
		if len(network.IPAM.Config) != 2 {
			return fmt.Errorf("Bad value for IPAM configuration count: %d", len(network.IPAM.Config))
		}
		if network.IPAM.Config[1].Subnet != "fd00::1/64" {
			return fmt.Errorf("Bad value for attribute 'subnet': %v", network.IPAM.Config[1].Subnet)
		}
		return nil
	}
}

func TestAccDockerNetwork_labels(t *testing.T) {
	var n network.Inspect
	resourceName := "docker_network.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_network", "testAccDockerNetworkLabelsConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccNetwork(resourceName, &n),
					testCheckLabelMap(resourceName, "labels",
						map[string]string{
							"com.docker.compose.network": "foo",
							"com.docker.compose.project": "test",
						},
					),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
