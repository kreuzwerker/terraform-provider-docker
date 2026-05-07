package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerConfig_basic(t *testing.T) {
	ctx := context.Background()
	var c swarm.Config

	testCheckConfigInspect := func(*terraform.State) error {
		if c.Spec.Name == "" {
			return errors.New("Config Spec.Name is empty")
		}

		if len(c.Spec.Data) == 0 {
			return errors.New("Config Spec.Data is empty")
		}

		if len(c.Spec.Labels) != 0 {
			return fmt.Errorf("Config Spec.Labels is wrong: %v", c.Spec.Labels)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testCheckDockerConfigDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_config" "foo" {
					name = "foo-config"
					data = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_config.foo", "name", "foo-config"),
					resource.TestCheckResourceAttr("docker_config.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
					testAccServiceConfigCreated("docker_config.foo", &c),
					testCheckConfigInspect,
				),
			},
			{
				ResourceName:      "docker_config.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDockerConfig_basicUpdatable(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testCheckDockerConfigDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_config" "foo" {
					name 			 = "tftest-myconfig-${replace(timestamp(),":", ".")}"
					data 			 = "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="

					lifecycle {
						ignore_changes        = ["name"]
						create_before_destroy = true
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_config.foo", "data", "Ymxhc2RzYmxhYmxhMTI0ZHNkd2VzZA=="),
				),
			},
			{
				Config: `
				resource "docker_config" "foo" {
					name 			 = "tftest-myconfig2-${replace(timestamp(),":", ".")}"
					data 			 = "U3VuIDI1IE1hciAyMDE4IDE0OjQ2OjE5IENFU1QK"

					lifecycle {
						ignore_changes        = ["name"]
						create_before_destroy = true
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_config.foo", "data", "U3VuIDI1IE1hciAyMDE4IDE0OjQ2OjE5IENFU1QK"),
				),
			},
			{
				ResourceName:      "docker_config.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDockerConfig_labels(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testCheckDockerConfigDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_config" "foo" {
					name = "foo-config"
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
				Check: testCheckLabelMap("docker_config.foo", "labels",
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
func testCheckDockerConfigDestroy(ctx context.Context, s *terraform.State) error {
	client, err := testAccProvider.Meta().(*ProviderConfig).MakeClient(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "configs" {
			continue
		}

		id := rs.Primary.Attributes["id"]
		_, _, err := client.ConfigInspectWithRaw(ctx, id)

		if err == nil {
			return fmt.Errorf("Config with id '%s' still exists", id)
		}
		return nil
	}
	return nil
}

func testAccServiceConfigCreated(resourceName string, config *swarm.Config) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource with name '%s' not found in state", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := testAccProvider.Meta().(*ProviderConfig).MakeClient(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to create Docker client: %w", err)
		}
		inspectedConfig, _, err := client.ConfigInspectWithRaw(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Config with ID '%s': %w", rs.Primary.ID, err)
		}

		// we set the value to the pointer to be able to use the value
		// outside of the function
		*config = inspectedConfig
		return nil

	}
}

func TestResourceDockerConfigSchemaDataAndDataRawAreMutuallyExclusive(t *testing.T) {
	resourceSchema := resourceDockerConfig().Schema
	expectedExactlyOneOf := []string{"data", "data_raw"}

	if !reflect.DeepEqual(resourceSchema["data"].ExactlyOneOf, expectedExactlyOneOf) {
		t.Fatalf("expected data ExactlyOneOf to be %v, got %v", expectedExactlyOneOf, resourceSchema["data"].ExactlyOneOf)
	}

	if !reflect.DeepEqual(resourceSchema["data_raw"].ExactlyOneOf, expectedExactlyOneOf) {
		t.Fatalf("expected data_raw ExactlyOneOf to be %v, got %v", expectedExactlyOneOf, resourceSchema["data_raw"].ExactlyOneOf)
	}
}

func TestGetConfigDataBytes(t *testing.T) {
	testCases := []struct {
		name         string
		configValues map[string]interface{}
		expected     []byte
		hasError     bool
	}{
		{
			name: "uses base64 data",
			configValues: map[string]interface{}{
				"data": base64.StdEncoding.EncodeToString([]byte("config text")),
			},
			expected: []byte("config text"),
		},
		{
			name: "prefers data_raw when present",
			configValues: map[string]interface{}{
				"data":     base64.StdEncoding.EncodeToString([]byte("base64 value")),
				"data_raw": "raw value",
			},
			expected: []byte("raw value"),
		},
		{
			name: "returns error for invalid base64 data",
			configValues: map[string]interface{}{
				"data": "%%%invalid-base64%%%",
			},
			hasError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t, resourceDockerConfig().Schema, testCase.configValues)

			got, err := getConfigDataBytes(resourceData)
			if testCase.hasError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, testCase.expected) {
				t.Fatalf("expected %q, got %q", string(testCase.expected), string(got))
			}
		})
	}
}
