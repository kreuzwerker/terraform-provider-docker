package provider

import (
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

func TestAccDockerBuildxBuilder_AutoRecreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_buildx_builder", "testAccDockerBuildxBuilderAutoRecreate"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_buildx_builder.test_auto_recreate", "name"),
					resource.TestCheckResourceAttr("docker_buildx_builder.test_auto_recreate", "auto_recreate", "true"),
					resource.TestCheckResourceAttr("docker_buildx_builder.test_auto_recreate", "bootstrap", "true"),
				),
			},
		},
	})
}

func TestAccDockerBuildxBuilder_AutoRecreateDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_buildx_builder", "testAccDockerBuildxBuilderDockerContainer"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_buildx_builder.foo", "name"),
					// auto_recreate should default to false
					resource.TestCheckResourceAttr("docker_buildx_builder.foo", "auto_recreate", "false"),
				),
			},
		},
	})
}
