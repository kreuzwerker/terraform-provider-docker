package docker

import (
	"fmt"
	"testing"

	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccDockerSecret_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckDockerSecretDestroy,
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
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckDockerSecretDestroy,
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
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckDockerSecretDestroy,
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
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						spew.Dump(s.RootModule().Resources["docker_secret.foo"].Primary.Attributes)
						return nil
					},
					resource.TestCheckResourceAttr("docker_secret.foo", "labels.0.label", "test1"),
					resource.TestCheckResourceAttr("docker_secret.foo", "labels.0.value", "foo"),
					resource.TestCheckResourceAttr("docker_secret.foo", "labels.1.label", "test1"),
					resource.TestCheckResourceAttr("docker_secret.foo", "labels.1.value", "bar"),
				),
			},
		},
	})
}

/////////////
// Helpers
/////////////
func testCheckDockerSecretDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ProviderConfig).DockerClient
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "secrets" {
			continue
		}

		id := rs.Primary.Attributes["id"]
		_, _, err := client.SecretInspectWithRaw(context.Background(), id)

		if err == nil {
			return fmt.Errorf("Secret with id '%s' still exists", id)
		}
		return nil
	}
	return nil
}
