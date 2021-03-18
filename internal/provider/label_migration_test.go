package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestMigrateServiceLabelState_empty_labels(t *testing.T) {
	v0State := map[string]interface{}{
		"name": "volume-name",
		"task_spec": []interface{}{
			map[string]interface{}{
				"container_spec": []interface{}{
					map[string]interface{}{
						"image": "repo:tag",
						"mounts": []interface{}{
							map[string]interface{}{
								"target": "path/to/target",
								"type":   "bind",
								"volume_options": []interface{}{
									map[string]interface{}{},
								},
							},
						},
					},
				},
			},
		},
	}

	// first validate that we build that correctly
	v0Config := terraform.NewResourceConfigRaw(v0State)
	diags := resourceDockerServiceV0().Validate(v0Config)
	if diags.HasError() {
		t.Error("test precondition failed - attempt to migrate an invalid v0 config")
		return
	}

	v1State := migrateServiceLabels(v0State)
	v1Config := terraform.NewResourceConfigRaw(v1State)
	diags = resourceDockerService().Validate(v1Config)
	if diags.HasError() {
		fmt.Println(diags)
		t.Error("migrated service config is invalid")
		return
	}
}

func TestMigrateServiceLabelState_with_labels(t *testing.T) {
	v0State := map[string]interface{}{
		"name": "volume-name",
		"task_spec": []interface{}{
			map[string]interface{}{
				"container_spec": []interface{}{
					map[string]interface{}{
						"image": "repo:tag",
						"labels": map[string]interface{}{
							"type": "container",
							"env":  "dev",
						},
						"mounts": []interface{}{
							map[string]interface{}{
								"target": "path/to/target",
								"type":   "bind",
								"volume_options": []interface{}{
									map[string]interface{}{
										"labels": map[string]interface{}{
											"type": "mount",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"labels": map[string]interface{}{
			"foo": "bar",
			"env": "dev",
		},
	}

	// first validate that we build that correctly
	v0Config := terraform.NewResourceConfigRaw(v0State)
	diags := resourceDockerServiceV0().Validate(v0Config)
	if diags.HasError() {
		t.Error("test precondition failed - attempt to migrate an invalid v0 config")
		return
	}

	v1State := migrateServiceLabels(v0State)
	v1Config := terraform.NewResourceConfigRaw(v1State)
	diags = resourceDockerService().Validate(v1Config)
	if diags.HasError() {
		fmt.Println(diags)
		t.Error("migrated service config is invalid")
		return
	}
}
