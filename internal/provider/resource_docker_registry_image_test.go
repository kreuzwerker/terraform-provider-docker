package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerRegistryImageResource_build_insecure_registry(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15001/tftest-dockerregistryimage:1.0")
	wd, _ := os.Getwd()
	context := strings.ReplaceAll((filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_context")), "\\", "\\\\")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testBuildDockerRegistryImageNoKeepConfig"), "http://127.0.0.1:15001", pushOptions.Name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.foo", "sha256_digest"),
				),
			},
		},
		CheckDestroy: testDockerRegistryImageNotInRegistry(pushOptions),
	})
}

func TestAccDockerRegistryImageResource_buildAndKeep(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15000/tftest-dockerregistryimage:1.0")
	wd, _ := os.Getwd()
	context := strings.ReplaceAll(filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_context"), "\\", "\\\\")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testBuildDockerRegistryImageKeepConfig"), pushOptions.Registry, pushOptions.Name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.foo", "sha256_digest"),
				),
			},
		},
		// as the providerConfig obtained from testAccProvider.Meta().(*ProviderConfig)
		// is empty after the test the credetials are passed here manually
		CheckDestroy: testDockerRegistryImageInRegistry("testuser", "testpwd", pushOptions, true),
	})
}

func TestAccDockerRegistryImageResource_pushMissingImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testDockerRegistryImagePushMissingConfig"),
				ExpectError: regexp.MustCompile("An image does not exist locally"),
			},
		},
	})
}

func testDockerRegistryImageNotInRegistry(pushOpts internalPushImageOptions) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		providerConfig := testAccProvider.Meta().(*ProviderConfig)
		authConfig, _ := getAuthConfigForRegistry(pushOpts.Registry, providerConfig)
		digest, _ := getImageDigestWithFallback(pushOpts, normalizeRegistryAddress(pushOpts.Registry), authConfig.Username, authConfig.Password, true)
		if digest != "" {
			return fmt.Errorf("image found")
		}
		return nil
	}
}

func testDockerRegistryImageInRegistry(username, password string, pushOpts internalPushImageOptions, cleanup bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		digest, err := getImageDigestWithFallback(pushOpts, normalizeRegistryAddress(pushOpts.Registry), username, password, true)
		if err != nil || len(digest) < 1 {
			return fmt.Errorf("image '%s' with credentials('%s' - '%s') not found: %w", pushOpts.Name, username, password, err)
		}
		if cleanup {
			err := deleteDockerRegistryImage(pushOpts, normalizeRegistryAddress(pushOpts.Registry), digest, username, password, true, false)
			if err != nil {
				return fmt.Errorf("Unable to remove test image '%s': %w", pushOpts.Name, err)
			}
		}
		return nil
	}
}
