package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var bar = regexp.MustCompile(`\A[A-Za-z0-9_\+\.-]+:[A-Fa-f0-9]+\z`)

func TestAccDockerRegistryMultiarchImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_registry_multiarch_image", "testAccDockerMultiarchImageDataSourceConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.docker_registry_multiarch_image.foo", "sha256_digest", bar),
				),
			},
		},
	})
}
