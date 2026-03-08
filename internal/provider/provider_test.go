package provider

import (
	"context"
	"fmt"
	"os"
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
	var _ = New("dev")()
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
				ExpectError: regexp.MustCompile(`expected "registry_auth.0.address" to not be an empty string, got `),
			},
		},
	})
}

func TestAccDockerProvider_WithMultipleRegistryAuth(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15000/tftest-dockerregistryimage-testtest:1.0")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "provider", "testAccDockerProviderMultipleRegistryAuth"), pushOptions.Registry),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.docker_registry_image.foobar", "sha256_digest"),
				),
			},
		},
	})
}

func TestAccDockerProvider_WithDisabledRegistryAuth(t *testing.T) {
	pushOptions := createPushImageOptions("http://127.0.0.1:15002/tftest-dockerregistryimage-testtest:1.0")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "provider", "testAccDockerProviderDisabledRegistryAuth"), pushOptions.NormalizedRegistry),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.docker_registry_image.foobar", "sha256_digest"),
				),
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
		cmd = exec.Command("docker", "swarm", "init", "--advertise-addr", "127.0.0.1")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Docker swarm could not be initialized: %s", err)
		}
	}

	err := testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetContextHost_ValidContext(t *testing.T) {
	// Create a temporary directory to simulate Docker contexts
	tempDir := t.TempDir()
	contextName := "test-context"
	contextUUID := "1234-5678-91011"
	contextFilePath := fmt.Sprintf("%s/.docker/contexts/meta/%s/meta.json", tempDir, contextUUID)

	// Simulate a valid Docker context file
	contextData := `{
		"Name": "test-context",
		"Endpoints": {
			"docker": {
				"Host": "tcp://docker:2375"
			}
		}
	}`
	if err := os.MkdirAll(fmt.Sprintf("%s/.docker/contexts/meta/%s", tempDir, contextUUID), 0755); err != nil {
		t.Fatalf("Failed to create context directory: %s", err)
	}
	if err := os.WriteFile(contextFilePath, []byte(contextData), 0644); err != nil {
		t.Fatalf("Failed to write context file: %s", err)
	}

	// Test the function
	host, err := getContextHost(contextName, tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %s", err)
	}
	if host != "tcp://docker:2375" {
		t.Fatalf("Expected host 'tcp://docker:2375', got: %s", host)
	}
}

func TestGetContextHost_InvalidContext(t *testing.T) {
	// Create a temporary directory to simulate Docker contexts
	tempDir := t.TempDir()

	if err := os.MkdirAll(fmt.Sprintf("%s/.docker/contexts/meta/foobar", tempDir), 0755); err != nil {
		t.Fatalf("Failed to create context directory: %s", err)
	}

	// Test the function with a non-existent context
	_, err := getContextHost("non-existent-context", tempDir)
	if err == nil || err.Error() != "context 'non-existent-context' not found" {
		t.Fatalf("Expected error 'context 'non-existent-context' not found', got: %v", err)
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
