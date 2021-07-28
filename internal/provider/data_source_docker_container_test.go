package provider

import (
	"os/exec"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerContainerDataSource_withName(t *testing.T) {
	containerName := "tf-test-nginx"
	var containerId string

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			containerId = startContainerForTest(t, containerName)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_container", "testAccDockerContainerDataSourceWithName"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_container.foo", "name", containerName),
					resource.TestCheckResourceAttr("data.docker_container.foo", "id", containerId),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return removeContainerForTest(t, containerName)
		},
	})
}

func TestAccDockerContainerDataSource_withMissingName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_container", "testAccDockerContainerDataSourceWithName"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_container.foo", "name", "wee"),
				),
			},
		},
	})
}

func startContainerForTest(t *testing.T, containerName string) string {
	cmd := exec.Command("docker", "run", "-d", "--name", containerName, "nginx:latest")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to start container '%s': %s", containerName, err)
	}
	cmd = exec.Command("docker", "inspect", "--format='{{.ID}}", containerName)
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to start container '%s': %s", containerName, err)
	}
	return string(stdout)
}
func removeContainerForTest(t *testing.T, containerName string) error {
	cmd := exec.Command("docker", "rm", "--force", containerName)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to remove container '%s': %s", containerName, err)
	}
	return nil
}
