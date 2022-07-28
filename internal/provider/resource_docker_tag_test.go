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
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_tag", "testAccDockerTag"), "nginx:1.17.6@sha256:36b77d8bb27ffca25c7f6f53cadd059aca2747d46fb6ef34064e31727325784e", "nginx:new_tag"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_tag.foobar", "source_image_id", "sha256:f7bb5701a33c0e572ed06ca554edca1bee96cbbc1f76f3b01c985de7e19d0657"),
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
	if err := exec.Command("docker", "pull", "busybox:1.35.0@sha256:8cde9b8065696b65d7b7ffaefbab0262d47a5a9852bfd849799559d296d2e0cd").Run(); err != nil {
		t.Fatal(err)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_tag", "testAccDockerTag"), "nginx:1.17.6@sha256:36b77d8bb27ffca25c7f6f53cadd059aca2747d46fb6ef34064e31727325784e", "nginx:new_tag"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_tag.foobar", "source_image_id", "sha256:f7bb5701a33c0e572ed06ca554edca1bee96cbbc1f76f3b01c985de7e19d0657"),
				),
			},
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_tag", "testAccDockerTag"), "busybox:1.35.0@sha256:8cde9b8065696b65d7b7ffaefbab0262d47a5a9852bfd849799559d296d2e0cd", "nginx:new_tag"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_tag.foobar", "source_image_id", "sha256:d8c0f97fc6a6ac400e43342e67d06222b27cecdb076cbf8a87f3a2a25effe81c"),
				),
			},
		},
	})
}
