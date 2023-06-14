package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var manifestRegexSet = map[string]*regexp.Regexp{
	"sha256_digest": regexp.MustCompile(`\A[A-Za-z0-9_\+\.-]+:[A-Fa-f0-9]+\z`),
	"architecture":  regexp.MustCompile(`\A(?:amd|amd64|arm|arm64|386|ppc64le|s390x|unknown)\z`),
	"os":            regexp.MustCompile(`\A(?:linux|unknown)\z`),
	"media_type":    regexp.MustCompile(`\Aapplication\/vnd\..+\z`),
}

func TestAccDockerRegistryMultiarchImageDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_registry_multiarch_image", "testAccDockerMultiarchImageDataSourceConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchTypeSetElemNestedAttrs("data.docker_registry_multiarch_image.foo", "manifests.*", manifestRegexSet),
				),
			},
		},
	})
}

func TestAccDockerRegistryMultiarchImageDataSource_private(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_registry_multiarch_image", "testAccDockerMultiarchImageDataSourcePrivateConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchTypeSetElemNestedAttrs("data.docker_registry_multiarch_image.bar", "manifests.*", manifestRegexSet),
				),
			},
		},
	})
}
