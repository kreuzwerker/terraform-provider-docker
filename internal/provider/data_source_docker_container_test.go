package provider

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerContainerDataSource_withName(t *testing.T) {
	containerName := "tf-test-nginx"
	containerId := new(string)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			*containerId = startContainerForTest(t, containerName)
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
				Config:      loadTestConfiguration(t, DATA_SOURCE, "docker_container", "testAccDockerContainerDataSourceWithName"),
				ExpectError: regexp.MustCompile(`Could not find*`),
			},
		},
	})
}

func TestAccDockerContainerDataSource_withNameWildcard(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			startContainerForTest(t, "tf-test-nginx-wildcard-1")
			startContainerForTest(t, "tf-test-nginx-wildcard-2")
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      loadTestConfiguration(t, DATA_SOURCE, "docker_container", "testAccDockerContainerDataSourceWithNameWildcard"),
				ExpectError: regexp.MustCompile(`Found multiple containers*`),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			err1 := removeContainerForTest(t, "tf-test-nginx-wildcard-1")
			err2 := removeContainerForTest(t, "tf-test-nginx-wildcard-2")
			if err1 != nil || err2 != nil {
				return errors.New("error tearing down containers")
			}
			return nil
		},
	})
}

func startContainerForTest(t *testing.T, containerName string) string {
	cmd := exec.Command("docker", "run", "-d", "--name", containerName, "nginx:latest")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to start container '%s': %s", containerName, err)
	}
	cmd = exec.Command("docker", "inspect", "--format={{.ID}}", containerName)
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to start container '%s': %s", containerName, err)
	}
	return strings.TrimSpace(string(stdout))
}
func removeContainerForTest(t *testing.T, containerName string) error {
	cmd := exec.Command("docker", "rm", "--force", containerName)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to remove container '%s': %s", containerName, err)
	}
	return nil
}
