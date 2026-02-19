package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

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
	if out, err := exec.Command("docker", "version").CombinedOutput(); err != nil {
		t.Fatalf("Docker must be available: %s\n%s", err, strings.TrimSpace(string(out)))
	}

	ensureSwarmManager(t)

	err := testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatal(err)
	}
}

type dockerSwarmInfo struct {
	LocalNodeState   string `json:"LocalNodeState"`
	ControlAvailable bool   `json:"ControlAvailable"`
}

func ensureSwarmManager(t *testing.T) {
	t.Helper()

	getSwarmInfo := func() (dockerSwarmInfo, string, error) {
		out, err := exec.Command("docker", "info", "--format", "{{json .Swarm}}").CombinedOutput()
		trimmed := strings.TrimSpace(string(out))
		if err != nil {
			return dockerSwarmInfo{}, trimmed, err
		}
		var info dockerSwarmInfo
		if unmarshalErr := json.Unmarshal([]byte(trimmed), &info); unmarshalErr != nil {
			return dockerSwarmInfo{}, trimmed, unmarshalErr
		}
		return info, trimmed, nil
	}

	isManager := func(info dockerSwarmInfo) bool {
		return strings.EqualFold(info.LocalNodeState, "active") && info.ControlAvailable
	}

	info, raw, err := getSwarmInfo()
	if err == nil && isManager(info) {
		return
	}

	// If we're in a swarm but not a manager (e.g. worker), `docker node ls` and
	// swarm-scoped API calls will fail with "This node is not a swarm manager".
	// The easiest self-heal in CI is to leave and re-init a single-node swarm.
	_, _ = exec.Command("docker", "swarm", "leave", "--force").CombinedOutput()

	initAttempts := [][]string{
		{"docker", "swarm", "init", "--advertise-addr", "127.0.0.1"},
		{"docker", "swarm", "init"},
	}
	var lastInitOut string
	var lastInitErr error
	for _, args := range initAttempts {
		out, initErr := exec.Command(args[0], args[1:]...).CombinedOutput()
		lastInitOut = strings.TrimSpace(string(out))
		lastInitErr = initErr
		if initErr == nil {
			break
		}
		// If the node is still considered part of a swarm, retry after leaving.
		if strings.Contains(lastInitOut, "already part of a swarm") {
			_, _ = exec.Command("docker", "swarm", "leave", "--force").CombinedOutput()
		}
	}
	if lastInitErr != nil {
		t.Fatalf("Docker swarm could not be initialized: %s\n%s\n(docker info swarm: %s)", lastInitErr, lastInitOut, raw)
	}

	// Wait for Swarm to become manager-ready (can be slightly async on CI).
	var lastRaw string
	for i := 0; i < 20; i++ {
		current, currentRaw, infoErr := getSwarmInfo()
		if infoErr == nil {
			lastRaw = currentRaw
			if isManager(current) {
				return
			}
		}
		time.Sleep(time.Duration(100+(i*50)) * time.Millisecond)
	}

	t.Fatalf("Docker swarm did not become manager-ready (last swarm info: %s)", lastRaw)
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
