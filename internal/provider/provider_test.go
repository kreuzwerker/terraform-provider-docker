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

var (
	// testAccProvider is the "main" provider instance
	//
	// This Provider can be used in testing code for API calls without requiring
	// the use of saving and referencing specific ProviderFactories instances.
	//
	// testAccPreCheck(t) must be called before using this provider instance.
	testAccProvider *schema.Provider
	// providerFactories are used to instantiate a provider during acceptance testing.
	// The factory function will be invoked for every Terraform CLI command executed
	// to create a provider server to which the CLI can reattach.
	providerFactories map[string]func() (*schema.Provider, error)
)

func init() {
	testAccProvider = New("dev")()
	providerFactories = map[string]func() (*schema.Provider, error){
		"docker": func() (*schema.Provider, error) {
			return New("dev")(), nil
		},
	}
}

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

	err := testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
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
	provider             = "docker.private"
	name                 = "localhost:15000/helloworld:1.0"
	insecure_skip_verify = true
}
`
