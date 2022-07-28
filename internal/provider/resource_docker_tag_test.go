package provider

import (
	"fmt"
	"os/exec"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Things to test:
// * what happens when source_image is not available locally
// * wrong source_image/target_image names
// * update of source_image name
// * update of target_image name

func TestAccDockerTag_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_tag", "testAccDockerTag"), "nginx:1.17.6", "nginx:new_tag"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_tag.foobar", "source_image_id", "sha256:bfbe0b456268eb8d8cc7b4134ec0111587994831ee2a9d17ba9cacddb800f562"),
				),
			},
		},
	})
}

func TestAccDockerTag_no_local_image(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_tag", "testAccDockerTag"), "nginx:not_existent_on_machine", "nginx:new_tag"),
				ExpectError: regexp.MustCompile(`Error response from daemon: No such image: nginx:not_existent_on_machine`),
			},
		},
	})
}
func TestAccDockerTag_wrong_names(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_tag", "testAccDockerTag"), "foobar//nginx:not_existent_on_machine", "nginx:new_tag"),
				ExpectError: regexp.MustCompile(`failed to create docker tag: Error parsing reference: "foobar//nginx:not_existent_on_machine"`),
			},
		},
	})
}

func TestAccDockerTag_source_image_changed(t *testing.T) {
	if err := exec.Command("docker", "pull", "nginx:1.17.6").Run(); err != nil {
		t.Fatal(err)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_tag", "testAccDockerTag"), "nginx:1.17.6", "nginx:new_tag"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_tag.foobar", "source_image_id", "sha256:bfbe0b456268eb8d8cc7b4134ec0111587994831ee2a9d17ba9cacddb800f562"),
				),
			},
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_tag", "testAccDockerTag"), "busybox:1.35.0", "nginx:new_tag"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_tag.foobar", "source_image_id", "sha256:4e294bde6038c93e7cf8a9e94ce74f3f759a865a62e88bdc54945d0426e9aea6"),
				),
			},
		},
	})
}
