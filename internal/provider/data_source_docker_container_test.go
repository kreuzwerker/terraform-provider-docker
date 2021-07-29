package provider

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDockerContainerDataSource_withName(t *testing.T) {
	containerName := "tf-test-data-container"
	containerId := new(string)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			*containerId = getContainerId(t, containerName)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_container", "testAccDockerContainerDataSourceWithName"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_container.foo", "name", fmt.Sprintf("/%s", containerName)),
					resource.TestCheckResourceAttrPtr("data.docker_container.foo", "id", containerId),
				),
			},
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
				Config:      loadTestConfiguration(t, DATA_SOURCE, "docker_container", "testAccDockerContainerDataSourceWithMissingName"),
				ExpectError: regexp.MustCompile(`Could not find*`),
			},
		},
	})
}

func TestAccDockerContainerDataSource_withNameWildcard(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      loadTestConfiguration(t, DATA_SOURCE, "docker_container", "testAccDockerContainerDataSourceWithNameWildcard"),
				ExpectError: regexp.MustCompile(`Found multiple containers*`),
			},
		},
	})
}

func getContainerId(t *testing.T, containerName string) string {
	cmd := exec.Command("docker", "inspect", "--format={{.ID}}", containerName)
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to start container '%s': %s", containerName, err)
	}
	return strings.TrimSpace(string(stdout))
}
