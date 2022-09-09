package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerSecret_basic(t *testing.T) {
	ctx := context.Background()
	var s swarm.Secret

	testCheckSecretInspect := func(*terraform.State) error {
		if s.Spec.Name == "" {
			return fmt.Errorf("Secret Spec.Name is wrong: %v", s.Spec.Name)
		}

		if len(s.Spec.Labels) != 1 || !mapEquals("foo", "bar", s.Spec.Labels) {
			return fmt.Errorf("Secret Spec.Labels is wrong: %v", s.Spec.Labels)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
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
						label = "foo"
						value = "bar"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_secret.foo", "name", "foo-secret"),
					resource.TestCheckResourceAttr("docker_secret.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
					testAccServiceSecretCreated("docker_secret.foo", &s),
					testCheckSecretInspect,
				),
			},
		},
	})
}

func TestAccDockerSecret_basicUpdatable(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
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
						ignore_changes        = ["name"]
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
						ignore_changes        = ["name"]
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
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
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

// ///////////
// Helpers
// ///////////
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

func testAccServiceSecretCreated(resourceName string, secret *swarm.Secret) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource with name '%s' not found in state", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		inspectedSecret, _, err := client.SecretInspectWithRaw(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Secret with ID '%s': %w", rs.Primary.ID, err)
		}

		// we set the value to the pointer to be able to use the value
		// outside of the function
		*secret = inspectedSecret
		return nil

	}
}
