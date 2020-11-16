package docker

import (
	"fmt"
	"testing"

	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerSecret_basic(t *testing.T) {
	ctx := context.TODO()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testCheckDockerSecretDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_secret" "foo" {
					name = "foo-secret"
					data = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_secret.foo", "name", "foo-secret"),
					resource.TestCheckResourceAttr("docker_secret.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
				),
			},
		},
	})
}

func TestAccDockerSecret_basicUpdatable(t *testing.T) {
	ctx := context.TODO()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testCheckDockerSecretDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_secret" "foo" {
					name 			 = "tftest-mysecret-${replace(timestamp(),":", ".")}"
					data 			 = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_secret.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
				),
			},
			{
				Config: `
				resource "docker_secret" "foo" {
					name 			 = "tftest-mysecret2-${replace(timestamp(),":", ".")}"
					data 			 = "U3VuIDI1IE1hciAyMDE4IDE0OjUzOjIxIENFU1QK"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_secret.foo", "data", "U3VuIDI1IE1hciAyMDE4IDE0OjUzOjIxIENFU1QK"),
				),
			},
		},
	})
}

func TestAccDockerSecret_labels(t *testing.T) {
	ctx := context.TODO()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testCheckDockerSecretDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_secret" "foo" {
					name = "foo-secret"
					data = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="
					labels {
						label = "test1"
						value = "foo"
					}
					labels {
						label = "test2"
						value = "bar"
					}
				}
				`,
				Check: testCheckLabelMap("docker_secret.foo", "labels",
					map[string]string{
						"test1": "foo",
						"test2": "bar",
					},
				),
			},
		},
	})
}

/////////////
// Helpers
/////////////
func testCheckDockerSecretDestroy(ctx context.Context, s *terraform.State) error {
	client := testAccProvider.Meta().(*ProviderConfig).DockerClient
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "secrets" {
			continue
		}

		id := rs.Primary.Attributes["id"]
		_, _, err := client.SecretInspectWithRaw(ctx, id)

		if err == nil {
			return fmt.Errorf("Secret with id '%s' still exists", id)
		}
		return nil
	}
	return nil
}
