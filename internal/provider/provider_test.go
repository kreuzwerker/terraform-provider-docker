package provider

import (
	"context"
	"os/exec"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// providerFactories are used to instantiate a provider during acceptance testing.
// The factory function will be invoked for every Terraform CLI command executed
// to create a provider server to which the CLI can reattach.
var providerFactories = map[string]func() (*schema.Provider, error){
	"docker": func() (*schema.Provider, error) {
		return New("dev")(), nil
	},
}

// TODO mavogel: can't we just use the factory?
// how is this variable used in tests?
var testAccProvider *schema.Provider

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = New("dev")()
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestAccDockerProvider_WithIncompleteRegistryAuth(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDockerProviderWithIncompleteAuthConfig,
				ExpectError: regexp.MustCompile(`401 Unauthorized`),
			},
		},
	})
}

func testAccPreCheck(t *testing.T) {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Docker must be available: %s", err)
	}

	cmd = exec.Command("docker", "node", "ls")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("docker", "swarm", "init")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Docker swarm could not be initialized: %s", err)
		}
	}

	// TODO mvogel: how to implement this properly
	ctx := context.TODO()
	err := testAccProvider.Configure(ctx, terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatal(err)
	}
}

const testAccDockerProviderWithIncompleteAuthConfig = `
provider "docker" {
	alias = "private"
	registry_auth {
	  address  = ""
	  username = ""
	  password = ""
	}
}
data "docker_registry_image" "foobar" {
	provider = "docker.private"
	name = "localhost:15000/helloworld:1.0"
}
`
