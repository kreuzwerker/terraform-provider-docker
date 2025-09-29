package provider

import (
	"os/exec"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDockerBuildxBuilder_DockerContainerDriver(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_buildx_builder", "testAccDockerBuildxBuilderDockerContainer"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_buildx_builder.foo", "name"),
				),
			},
		},
	})
}

func TestAccDockerBuildxBuilder_OutOfBandDeletion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_buildx_builder", "testAccDockerBuildxBuilderDockerContainer"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_buildx_builder.foo", "name"),
				),
			},
			{
				// This step simulates the external deletion or non-existence of the resource
				PreConfig: func() {
					if err := exec.Command("docker", "buildx", "rm", "foo").Run(); err != nil {
						t.Fatalf("failed to delete resource externally: %v", err)
					}
				},
				PlanOnly: true,
				Config:   loadTestConfiguration(t, RESOURCE, "docker_buildx_builder", "testAccDockerBuildxBuilderDockerContainer"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_buildx_builder.foo", "name", "test"),
				),
				// We expect the plan to show that the resource will be recreated because it was deleted out-of-band
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
