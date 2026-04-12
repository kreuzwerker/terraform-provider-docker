package actiontests

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	providerpkg "github.com/terraform-providers/terraform-provider-docker/internal/provider"
)

func TestAccDockerRegistryImageTags_DockerHub(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6RegistryProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
provider "docker" {}

data "docker_registry_image_tags" "foo" {
  name = "alpine"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.docker_registry_image_tags.foo", "id"),
					resource.TestCheckResourceAttrSet("data.docker_registry_image_tags.foo", "tags.#"),
					testCheckDockerRegistryImageTagsContains("data.docker_registry_image_tags.foo", "latest"),
				),
			},
		},
	})
}

func protoV6RegistryProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"docker": func() (tfprotov6.ProviderServer, error) {
			ctx := context.Background()

			upgradedSDKProvider, err := tf5to6server.UpgradeServer(ctx, providerpkg.New("test")().GRPCProvider)
			if err != nil {
				return nil, err
			}

			frameworkProviderServerFactory := providerserver.NewProtocol6WithError(providerpkg.NewFrameworkProvider("test")())

			muxServer, err := tf6muxserver.NewMuxServer(
				ctx,
				func() tfprotov6.ProviderServer {
					return upgradedSDKProvider
				},
				func() tfprotov6.ProviderServer {
					frameworkProviderServer, frameworkErr := frameworkProviderServerFactory()
					if frameworkErr != nil {
						panic(frameworkErr)
					}

					return frameworkProviderServer
				},
			)
			if err != nil {
				return nil, err
			}

			return muxServer.ProviderServer(), nil
		},
	}
}

func testCheckDockerRegistryImageTagsContains(resourceName string, expectedTag string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %q not found in state", resourceName)
		}

		count, err := strconv.Atoi(rs.Primary.Attributes["tags.#"])
		if err != nil {
			return fmt.Errorf("failed to parse tag count: %w", err)
		}
		if count == 0 {
			return fmt.Errorf("expected at least one tag, found none")
		}

		for i := 0; i < count; i++ {
			if rs.Primary.Attributes[fmt.Sprintf("tags.%d", i)] == expectedTag {
				return nil
			}
		}

		return fmt.Errorf("expected tag %q not found in state: %#v", expectedTag, rs.Primary.Attributes)
	}
}
