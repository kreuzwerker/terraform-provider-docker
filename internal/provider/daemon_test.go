package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const unknownValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

func testDaemonResource() *schema.Resource {
	return withDaemon(&schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
		ReadContext: func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
			return nil
		},
	})
}

func testProviderConfig(host string) *ProviderConfig {
	return &ProviderConfig{
		DefaultConfig: &Config{
			Host:    host,
			SSHOpts: []string{"-o", "StrictHostKeyChecking=no"},
			Ca:      "provider-ca",
		},
		AuthConfigs: &AuthConfigs{},
	}
}

func TestResolve_noBlockUsesProvider(t *testing.T) {
	c := testProviderConfig("unix:///var/run/docker.sock")
	d := schema.TestResourceDataRaw(t, testDaemonResource().Schema, map[string]interface{}{
		"name": "x",
	})

	config, auth, err := c.resolve(d)
	if err != nil {
		t.Fatal(err)
	}
	if config.Host != "unix:///var/run/docker.sock" || config.Ca != "provider-ca" {
		t.Fatalf("expected provider config, got %#v", config)
	}
	if auth != c.AuthConfigs {
		t.Fatal("expected provider auth configs")
	}
}

func TestResolve_nilResourceUsesProvider(t *testing.T) {
	c := testProviderConfig("unix:///var/run/docker.sock")

	config, _, err := c.resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	if config.Host != "unix:///var/run/docker.sock" {
		t.Fatalf("expected provider host, got %q", config.Host)
	}
}

func TestResolve_blockReplacesProvider(t *testing.T) {
	c := testProviderConfig("unix:///var/run/docker.sock")
	d := schema.TestResourceDataRaw(t, testDaemonResource().Schema, map[string]interface{}{
		"name": "x",
		"daemon": []interface{}{
			map[string]interface{}{
				"host":     "ssh://ops@10.0.0.5:22",
				"ssh_opts": []interface{}{"-o", "StrictHostKeyChecking=accept-new"},
				"registry_auth": []interface{}{
					map[string]interface{}{
						"address":  "ghcr.io",
						"username": "ci",
						"password": "secret",
					},
				},
			},
		},
	})

	config, auth, err := c.resolve(d)
	if err != nil {
		t.Fatal(err)
	}
	if config.Host != "ssh://ops@10.0.0.5:22" {
		t.Fatalf("expected daemon host, got %q", config.Host)
	}
	if config.Ca != "" {
		t.Fatalf("expected no inherited CA, got %q", config.Ca)
	}
	if len(config.SSHOpts) != 2 || config.SSHOpts[1] != "StrictHostKeyChecking=accept-new" {
		t.Fatalf("expected daemon ssh_opts, got %#v", config.SSHOpts)
	}
	if got := auth.Configs["ghcr.io"]; got.Username != "ci" || got.Password != "secret" {
		t.Fatalf("expected daemon registry auth, got %#v", auth.Configs)
	}
}

func TestConfigHash_includesSSHOptsInOrder(t *testing.T) {
	base := Config{Host: "ssh://ops@10.0.0.5:22"}
	a := base
	a.SSHOpts = []string{"-o", "A=1", "-o", "B=2"}
	b := base
	b.SSHOpts = []string{"-o", "B=2", "-o", "A=1"}

	if base.Hash() == a.Hash() {
		t.Fatal("ssh_opts must be part of the hash")
	}
	if a.Hash() == b.Hash() {
		t.Fatal("ssh_opts order must be part of the hash")
	}
}

func TestDaemonCustomizeDiff(t *testing.T) {
	cases := map[string]struct {
		state        map[string]string
		config       map[string]interface{}
		providerHost string
		replace      bool
	}{
		"adding a block adopts the resource": {
			state: map[string]string{"name": "x"},
			config: map[string]interface{}{
				"name":   "x",
				"daemon": []interface{}{map[string]interface{}{"host": "ssh://a"}},
			},
			replace: false,
		},
		"changing the host replaces": {
			state: map[string]string{"name": "x", "daemon.#": "1", "daemon.0.host": "ssh://a"},
			config: map[string]interface{}{
				"name":   "x",
				"daemon": []interface{}{map[string]interface{}{"host": "ssh://b"}},
			},
			replace: true,
		},
		"changing ssh_opts updates in place": {
			state: map[string]string{"name": "x", "daemon.#": "1", "daemon.0.host": "ssh://a"},
			config: map[string]interface{}{
				"name": "x",
				"daemon": []interface{}{map[string]interface{}{
					"host":     "ssh://a",
					"ssh_opts": []interface{}{"-o", "ConnectTimeout=5"},
				}},
			},
			replace: false,
		},
		"unknown host replaces": {
			state: map[string]string{"name": "x", "daemon.#": "1", "daemon.0.host": "ssh://a"},
			config: map[string]interface{}{
				"name":   "x",
				"daemon": []interface{}{map[string]interface{}{"host": unknownValue}},
			},
			replace: true,
		},
		"switching to a context replaces": {
			state: map[string]string{"name": "x", "daemon.#": "1", "daemon.0.host": "ssh://a"},
			config: map[string]interface{}{
				"name":   "x",
				"daemon": []interface{}{map[string]interface{}{"context": "box-b"}},
			},
			replace: true,
		},
		"removing the block keeps the resource on the same host": {
			state:        map[string]string{"name": "x", "daemon.#": "1", "daemon.0.host": "ssh://a"},
			config:       map[string]interface{}{"name": "x"},
			providerHost: "ssh://a",
			replace:      false,
		},
		"removing the block moves the resource to the provider host": {
			state:        map[string]string{"name": "x", "daemon.#": "1", "daemon.0.host": "ssh://a"},
			config:       map[string]interface{}{"name": "x"},
			providerHost: "unix:///var/run/docker.sock",
			replace:      true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := testDaemonResource()
			state := &terraform.InstanceState{ID: "id", Attributes: tc.state}

			diff, err := r.SimpleDiff(context.Background(), state, terraform.NewResourceConfigRaw(tc.config), testProviderConfig(tc.providerHost))
			if err != nil {
				t.Fatal(err)
			}

			if got := diff != nil && diff.RequiresNew(); got != tc.replace {
				t.Fatalf("expected replace=%t, got %t (diff: %#v)", tc.replace, got, diff)
			}
		})
	}
}

func testAccDaemonHost() string {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return host
	}
	return "unix:///var/run/docker.sock"
}

func testAccDaemonConfig(t *testing.T, name string) string {
	return fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_container", name), testAccDaemonHost())
}

func testCheckContainerID(c *container.InspectResponse, id *string, same bool) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if same && c.ID != *id {
			return fmt.Errorf("expected container %s to be kept, got %s", *id, c.ID)
		}
		if !same && c.ID == *id {
			return fmt.Errorf("expected container %s to be replaced", *id)
		}
		*id = c.ID
		return nil
	}
}

func testCheckContainerGone(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := testAccProvider.Meta().(*ProviderConfig).MakeClient(context.Background(), nil)
		if err != nil {
			return err
		}
		if _, err := client.ContainerInspect(context.Background(), name); err == nil {
			return fmt.Errorf("container %s still exists", name)
		}
		return nil
	}
}

// The provider points at an unreachable host, so every API call (create,
// refresh, the data source read, destroy) has to go through the daemon block.
func TestAccDockerContainer_daemon(t *testing.T) {
	var c container.InspectResponse
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDaemonConfig(t, "testAccDockerContainerDaemonConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					resource.TestCheckResourceAttr("docker_container.foo", "daemon.0.host", testAccDaemonHost()),
					resource.TestCheckResourceAttr("data.docker_network.bridge", "driver", "bridge"),
				),
			},
		},
	})
}

// Removing a host and its resources in one apply: the provider has no way to
// reach the daemon, so Delete has to use the host stored in state.
func TestAccDockerContainer_daemonRemoveHost(t *testing.T) {
	var c container.InspectResponse
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDaemonConfig(t, "testAccDockerContainerDaemonConfig"),
				Check:  testAccContainerRunning("docker_container.foo", &c),
			},
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerDaemonRemovedConfig"),
				Check:  testCheckContainerGone("tf-test-daemon"),
			},
		},
	})
}

// Replacing the host in one apply: the host is unknown during plan, refresh
// uses the host in state, and the resources are replaced on the new host.
func TestAccDockerContainer_daemonReplaceHost(t *testing.T) {
	var c container.InspectResponse
	var id string
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDaemonConfig(t, "testAccDockerContainerDaemonReplaceHostConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheckContainerID(&c, &id, false),
				),
			},
			{
				Config: testAccDaemonConfig(t, "testAccDockerContainerDaemonReplaceHostConfig"),
				Taint:  []string{"terraform_data.vm"},
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheckContainerID(&c, &id, false),
				),
			},
		},
	})
}

// Adding a daemon block to existing resources adopts them, changing ssh_opts
// updates in place, and removing the block again (same host) keeps them.
func TestAccDockerContainer_daemonAdopt(t *testing.T) {
	var c container.InspectResponse
	var id string
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerDaemonAdoptConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheckContainerID(&c, &id, false),
				),
			},
			{
				Config: testAccDaemonConfig(t, "testAccDockerContainerDaemonAdoptedConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheckContainerID(&c, &id, true),
				),
			},
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerDaemonAdoptConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheckContainerID(&c, &id, true),
				),
			},
		},
	})
}
