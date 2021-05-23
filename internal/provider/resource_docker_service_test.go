package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// ----------------------------------------
// -----------    UNIT  TESTS   -----------
// ----------------------------------------
func TestMigrateServiceV1ToV2_empty_restart_policy_and_auth(t *testing.T) {
	v1State := map[string]interface{}{
		"name": "volume-name",
		"task_spec": []interface{}{
			map[string]interface{}{
				"container_spec": []interface{}{
					map[string]interface{}{
						"image":             "repo:tag",
						"stop_grace_period": "10s",
					},
				},
			},
		},
	}

	// first validate that we build that correctly
	v1Config := terraform.NewResourceConfigRaw(v1State)
	diags := resourceDockerServiceV1().Validate(v1Config)
	if diags.HasError() {
		t.Error("test precondition failed - attempt to migrate an invalid v1 config")
		return
	}

	ctx := context.Background()
	v2State, _ := resourceDockerServiceStateUpgradeV2(ctx, v1State, nil)
	v2Config := terraform.NewResourceConfigRaw(v2State)
	diags = resourceDockerService().Validate(v2Config)
	if diags.HasError() {
		fmt.Println(diags)
		t.Error("migrated service config is invalid")
		return
	}
}
func TestMigrateServiceV1ToV2_with_restart_policy(t *testing.T) {
	v1State := map[string]interface{}{
		"name": "volume-name",
		"task_spec": []interface{}{
			map[string]interface{}{
				"container_spec": []interface{}{
					map[string]interface{}{
						"image":             "repo:tag",
						"stop_grace_period": "10s",
					},
				},
				"restart_policy": map[string]interface{}{
					"condition":    "on-failure",
					"delay":        "3s",
					"max_attempts": 4,
					"window":       "10s",
				},
			},
		},
	}

	// first validate that we build that correctly
	v1Config := terraform.NewResourceConfigRaw(v1State)
	diags := resourceDockerServiceV1().Validate(v1Config)
	if diags.HasError() {
		t.Error("test precondition failed - attempt to migrate an invalid v1 config")
		return
	}

	ctx := context.Background()
	v2State, _ := resourceDockerServiceStateUpgradeV2(ctx, v1State, nil)
	v2Config := terraform.NewResourceConfigRaw(v2State)
	diags = resourceDockerService().Validate(v2Config)
	if diags.HasError() {
		fmt.Println(diags)
		t.Error("migrated service config is invalid")
		return
	}
}

func TestMigrateServiceV1ToV2_with_auth(t *testing.T) {
	v1State := map[string]interface{}{
		"auth": map[string]interface{}{
			"server_address": "docker-reg.acme.com",
			"username":       "user",
			"password":       "pass",
		},
		"name": "volume-name",
		"task_spec": []interface{}{
			map[string]interface{}{
				"container_spec": []interface{}{
					map[string]interface{}{
						"image":             "repo:tag",
						"stop_grace_period": "10s",
					},
				},
			},
		},
	}

	// first validate that we build that correctly
	v1Config := terraform.NewResourceConfigRaw(v1State)
	diags := resourceDockerServiceV1().Validate(v1Config)
	if diags.HasError() {
		t.Error("test precondition failed - attempt to migrate an invalid v1 config")
		return
	}

	ctx := context.Background()
	v2State, _ := resourceDockerServiceStateUpgradeV2(ctx, v1State, nil)
	v2Config := terraform.NewResourceConfigRaw(v2State)
	diags = resourceDockerService().Validate(v2Config)
	if diags.HasError() {
		fmt.Println(diags)
		t.Error("migrated service config is invalid")
		return
	}
}

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

func TestDockerSecretFromRegistryAuth_basic(t *testing.T) {
	authConfigs := make(map[string]types.AuthConfig)
	authConfigs["https://repo.my-company.com:8787"] = types.AuthConfig{
		Username:      "myuser",
		Password:      "mypass",
		Email:         "",
		ServerAddress: "repo.my-company.com:8787",
	}

	foundAuthConfig := fromRegistryAuth("repo.my-company.com:8787/my_image", authConfigs)
	checkAttribute(t, "Username", foundAuthConfig.Username, "myuser")
	checkAttribute(t, "Password", foundAuthConfig.Password, "mypass")
	checkAttribute(t, "Email", foundAuthConfig.Email, "")
	checkAttribute(t, "ServerAddress", foundAuthConfig.ServerAddress, "repo.my-company.com:8787")
}

func TestDockerSecretFromRegistryAuth_multiple(t *testing.T) {
	authConfigs := make(map[string]types.AuthConfig)
	authConfigs["https://repo.my-company.com:8787"] = types.AuthConfig{
		Username:      "myuser",
		Password:      "mypass",
		Email:         "",
		ServerAddress: "repo.my-company.com:8787",
	}
	authConfigs["https://nexus.my-fancy-company.com"] = types.AuthConfig{
		Username:      "myuser33",
		Password:      "mypass123",
		Email:         "test@example.com",
		ServerAddress: "nexus.my-fancy-company.com",
	}

	foundAuthConfig := fromRegistryAuth("nexus.my-fancy-company.com/the_image", authConfigs)
	checkAttribute(t, "Username", foundAuthConfig.Username, "myuser33")
	checkAttribute(t, "Password", foundAuthConfig.Password, "mypass123")
	checkAttribute(t, "Email", foundAuthConfig.Email, "test@example.com")
	checkAttribute(t, "ServerAddress", foundAuthConfig.ServerAddress, "nexus.my-fancy-company.com")

	foundAuthConfig = fromRegistryAuth("alpine:3.1", authConfigs)
	checkAttribute(t, "Username", foundAuthConfig.Username, "")
	checkAttribute(t, "Password", foundAuthConfig.Password, "")
	checkAttribute(t, "Email", foundAuthConfig.Email, "")
	checkAttribute(t, "ServerAddress", foundAuthConfig.ServerAddress, "")
}

func checkAttribute(t *testing.T, name, actual, expected string) {
	if actual != expected {
		t.Fatalf("bad authconfig attribute for '%q'\nExpected: %s\n     Got: %s", name, expected, actual)
	}
}

func TestDockerImageNameSuppress(t *testing.T) {
	suppressFunc := suppressIfSHAwasAdded()
	old := ""
	new := "alpine3.1"
	suppress := suppressFunc("k", old, new, nil)
	if suppress {
		t.Fatalf("Expected no suppress for \n\told '%s' \n\tnew '%s'", old, new)
	}

	old = "127.0.0.1:15000/tftest-service:v1"
	new = "127.0.0.1:15000/tftest-service:v1@sha256:74d04f400723d9770187ee284255d1eb556f3d51700792fb2bfd6ab13da50981"
	suppress = suppressFunc("k", old, new, nil)
	if !suppress {
		t.Fatalf("Expected suppress for \n\told '%s' \n\tnew '%s'", old, new)
	}

	old = "127.0.0.1:15000/tftest-service:latest@sha256:74d04f400723d9770187ee284255d1eb556f3d51700792fb2bfd6ab13da50981"
	new = "127.0.0.1:15000/tftest-service"
	suppress = suppressFunc("k", old, new, nil)
	if !suppress {
		t.Fatalf("Expected suppress for \n\told '%s' \n\tnew '%s'", old, new)
	}

	old = "127.0.0.1:15000/tftest-service:latest"
	new = "127.0.0.1:15000/tftest-service:latest@sha256:74d04f400723d9770187ee284255d1eb556f3d51700792fb2bfd6ab13da50981"
	suppress = suppressFunc("k", old, new, nil)
	if suppress {
		t.Fatalf("Expected no suppress for \n\told '%s' \n\tnew '%s'", old, new)
	}

	old = "127.0.0.1:15000/tftest-service"
	new = "127.0.0.1:15000/tftest-service:latest@sha256:74d04f400723d9770187ee284255d1eb556f3d51700792fb2bfd6ab13da50981"
	suppress = suppressFunc("k", old, new, nil)
	if suppress {
		t.Fatalf("Expected no suppress for \n\told '%s' \n\tnew '%s'", old, new)
	}

	old = "127.0.0.1:15000/tftest-service:v1"
	new = "127.0.0.1:15000/tftest-service:v2@sha256:ed8e15d68bb13e3a04abddc295f87d2a8b7d849d5ff91f00dbdd66dc10fd8aac"
	suppress = suppressFunc("k", old, new, nil)
	if suppress {
		t.Fatalf("Expected no suppress for image tag update from \n\told '%s' \n\tnew '%s'", old, new)
	}

	old = "127.0.0.1:15000/tftest-service:v1@sha256:74d04f400723d9770187ee284255d1eb556f3d51700792fb2bfd6ab13da50981"
	new = "127.0.0.1:15000/tftest-service:v2@sha256:74d04f400723d9770187ee284255d1eb556f3d51700792fb2bfd6ab13da50981"
	suppress = suppressFunc("k", old, new, nil)
	if suppress {
		t.Fatalf("Expected no suppress for image tag update from \n\told '%s' \n\tnew '%s'", old, new)
	}

	old = "127.0.0.1:15000/tftest-service:latest@sha256:74d04f400723d9770187ee284255d1eb556f3d51700792fb2bfd6ab13da50981"
	new = "127.0.0.1:15000/tftest-service:latest@sha256:c9d1055182f0607632b7d859d2f220126fb1c0d10aedc4451817840b30c1af86"
	suppress = suppressFunc("k", old, new, nil)
	if suppress {
		t.Fatalf("Expected no suppress for image digest update from \n\told '%s' \n\tnew '%s'", old, new)
	}

	old = "127.0.0.1:15000/tftest-service:v3@sha256:74d04f400723d9770187ee284255d1eb556f3d51700792fb2bfd6ab13da50981"
	new = "127.0.0.1:15000/tftest-service:latest@sha256:c9d1055182f0607632b7d859d2f220126fb1c0d10aedc4451817840b30c1af86"
	suppress = suppressFunc("k", old, new, nil)
	if suppress {
		t.Fatalf("Expected no suppress for image tag but no digest update from \n\told '%s' \n\tnew '%s'", old, new)
	}

	old = "127.0.0.1:15000/tftest-service@sha256:74d04f400723d9770187ee284255d1eb556f3d51700792fb2bfd6ab13da50981"
	new = "127.0.0.1:15000/tftest-service@sha256:c9d1055182f0607632b7d859d2f220126fb1c0d10aedc4451817840b30c1af86"
	suppress = suppressFunc("k", old, new, nil)
	if suppress {
		t.Fatalf("Expected no suppress for image tag but no digest update from \n\told '%s' \n\tnew '%s'", old, new)
	}
}

// ----------------------------------------
// ----------- ACCEPTANCE TESTS -----------
// ----------------------------------------
// Fire and Forget
var serviceIDRegex = regexp.MustCompile(`[A-Za-z0-9_\+\.-]+`)

func TestAccDockerService_minimalSpec(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				provider "docker" {
					registry_auth {
						address = "127.0.0.1:15000"
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec {
							image 			  = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
						}
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1@sha256.*`)),
				),
			},
			{
				ResourceName:      "docker_service.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerService_fullSpec(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				provider "docker" {
					registry_auth {
						address = "127.0.0.1:15000"
					}
				}

				resource "docker_volume" "test_volume" {
					name = "tftest-volume"
				}

				resource "docker_config" "service_config" {
					name = "tftest-full-myconfig"
					data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
				}

				resource "docker_secret" "service_secret" {
					name = "tftest-mysecret"
					data = "ewogICJrZXkiOiAiUVdFUlRZIgp9"
				}

				resource "docker_network" "test_network" {
					name   = "tftest-network"
					driver = "overlay"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-basic"

					labels {
						label = "servicelabel"
						value = "true"
					}

					task_spec {
						container_spec {
							image = "127.0.0.1:15000/tftest-service:v1"

							labels {
								label = "foo"
								value = "bar"
							}

							command  = ["ls"]
							args     = ["-las"]
							hostname = "my-fancy-service"

							env = {
								MYFOO = "BAR"
								URI   = "/api-call?param1=value1"
							}

							dir    = "/root"
							user   = "root"
							groups = ["docker", "foogroup"]

							privileges {
								se_linux_context {
									disable = true
									user    = "user-label"
									role    = "role-label"
									type    = "type-label"
									level   = "level-label"
								}
							}

							read_only = true

							mounts {
								target      = "/mount/test"
								source      = docker_volume.test_volume.name
								type        = "volume"
								read_only   = true

								volume_options {
									no_copy = true
									labels {
										label = "foo"
										value = "bar"
									}
									driver_name = "random-driver"
									driver_options = {
										op1 = "val1"
									}
								}
							}

							stop_signal       = "SIGTERM"
							stop_grace_period = "10s"

							healthcheck {
								test     = ["CMD", "curl", "-f", "localhost:8080/health"]
								interval = "5s"
								timeout  = "2s"
								retries  = 4
							}

							hosts {
								host = "testhost"
								ip   = "10.0.1.0"
							}

							dns_config {
								nameservers = ["8.8.8.8"]
								search      = ["example.org"]
								options     = ["timeout:3"]
							}

							secrets {
								secret_id   = docker_secret.service_secret.id
								secret_name = docker_secret.service_secret.name
								file_name   = "/secrets.json"
								file_uid    = "0"
								file_gid    = "0"
								file_mode   = 0777
							}

							configs {
								config_id   = docker_config.service_config.id
								config_name = docker_config.service_config.name
								file_name = "/configs.json"
							}
						}

						resources {
							limits {
								nano_cpus    = 1000000
								memory_bytes = 536870912
							}
						}

						restart_policy {
							condition    = "on-failure"
							delay        = "3s"
							max_attempts = 4
							window       = "10s"
						}

						placement {
							constraints = [
								"node.role==manager",
							]

							prefs = [
								"spread=node.role.manager",
							]

							platforms {
								architecture = "amd64"
								os 			 = "linux"
							}

							max_replicas = 2
						}

						force_update = 0
						runtime      = "container"
						networks     = [docker_network.test_network.id]

						log_driver {
							name = "json-file"

							options = {
								max-size = "10m"
								max-file = "3"
							}
						}
					}

					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 2
						delay             = "10s"
						failure_action    = "pause"
						monitor           = "5s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					rollback_config {
						parallelism       = 2
						delay             = "5ms"
						failure_action    = "pause"
						monitor           = "10h"
						max_failure_ratio = "0.9"
						order             = "stop-first"
					}

					endpoint_spec {
						mode = "vip"

						ports {
							name           = "random"
							protocol       = "tcp"
							target_port    = "8080"
							published_port = "8080"
							publish_mode   = "ingress"
						}
					}
				}

				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					testCheckLabelMap("docker_service.foo", "labels", map[string]string{"servicelabel": "true"}),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1.*`)),
					testCheckLabelMap("docker_service.foo", "task_spec.0.container_spec.0.labels", map[string]string{"foo": "bar"}),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.command.0", "ls"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.args.0", "-las"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hostname", "my-fancy-service"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.env.MYFOO", "BAR"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.env.URI", "/api-call?param1=value1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dir", "/root"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.user", "root"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.groups.0", "docker"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.groups.1", "foogroup"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.privileges.0.se_linux_context.0.disable", "true"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.privileges.0.se_linux_context.0.user", "user-label"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.privileges.0.se_linux_context.0.role", "role-label"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.privileges.0.se_linux_context.0.type", "type-label"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.privileges.0.se_linux_context.0.level", "level-label"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.read_only", "true"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.0.target", "/mount/test"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.0.source", "tftest-volume"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.0.type", "volume"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.0.read_only", "true"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.0.volume_options.0.no_copy", "true"),
					testCheckLabelMap("docker_service.foo", "task_spec.0.container_spec.0.mounts.0.volume_options.0.labels", map[string]string{"foo": "bar"}),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.0.volume_options.0.driver_name", "random-driver"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.0.volume_options.0.driver_options.op1", "val1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.stop_signal", "SIGTERM"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.stop_grace_period", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "5s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.0.host", "testhost"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.0.ip", "10.0.1.0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dns_config.0.nameservers.0", "8.8.8.8"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dns_config.0.search.0", "example.org"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dns_config.0.options.0", "timeout:3"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.resources.0.limits.0.nano_cpus", "1000000"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.resources.0.limits.0.memory_bytes", "536870912"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.restart_policy.0.condition", "on-failure"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.restart_policy.0.delay", "3s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.restart_policy.0.max_attempts", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.restart_policy.0.window", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.placement.0.constraints.0", "node.role==manager"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.placement.0.prefs.0", "spread=node.role.manager"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.placement.0.max_replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.force_update", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.networks.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.name", "json-file"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-file", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-size", "10m"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "5s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.parallelism", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.delay", "5ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.monitor", "10h"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.max_failure_ratio", "0.9"),
					resource.TestCheckResourceAttr("docker_service.foo", "rollback_config.0.order", "stop-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.mode", "vip"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.name", "random"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.protocol", "tcp"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.target_port", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.published_port", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.publish_mode", "ingress"),
				),
			},
			{
				ResourceName:      "docker_service.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerService_partialReplicationConfig(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				provider "docker" {
					registry_auth {
						address = "127.0.0.1:15000"
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec {
							image             = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
						}
					}
					mode {}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1@sha256.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "1"),
				),
			},
			{
				Config: `
				provider "docker" {
					registry_auth {
						address = "127.0.0.1:15000"
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec {
							image             = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
						}
					}
					mode {
						replicated {}
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1@sha256.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "1"),
				),
			},
			{
				Config: `
				provider "docker" {
					registry_auth {
						address = "127.0.0.1:15000"
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec {
							image             = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
						}
					}
					mode {
						replicated {
							replicas = 2
						}
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1@sha256.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
				),
			},
			{
				ResourceName:      "docker_service.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerService_globalReplicationMode(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				provider "docker" {
					registry_auth {
						address = "127.0.0.1:15000"
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec {
							image             = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
						}
					}
					mode {
						global = true
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1@sha256.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.global", "true"),
				),
			},
			{
				ResourceName:      "docker_service.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerService_ConflictingGlobalAndReplicated(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
						}
					}
					mode {
						replicated {
							replicas = 2
						}
						global = true
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.*conflicts with.*`),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerService_ConflictingGlobalModeAndConverge(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				provider "docker" {
					registry_auth {
						address = "127.0.0.1:15000"
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
						}
					}
					mode {
						global = true
					}
					converge_config {
						delay    = "7s"
						timeout  = "10s"
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.*conflicts with.*`),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

// Converging tests
func TestAccDockerService_privateImageConverge(t *testing.T) {
	registry := "127.0.0.1:15000"
	image := "127.0.0.1:15000/tftest-service:v1"
	ctx := context.Background()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					provider "docker" {
						registry_auth {
							address = "%s"
						}
					}

					resource "docker_service" "foo" {
						name     = "tftest-service-foo"
						task_spec {
							container_spec {
								image             = "%s"
								stop_grace_period = "10s"
								
							}
						}
						mode {
							replicated {
								replicas = 2
							}
						}

						converge_config {
							delay    = "7s"
							timeout  = "3m"
						}
					}
				`, registry, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-foo"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1@sha256.*`)),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerService_updateMultiplePropertiesConverge(t *testing.T) {
	// Step 1
	configData := "ewogICJwcmVmaXgiOiAiMTIzIgp9"
	secretData := "ewogICJrZXkiOiAiUVdFUlRZIgp9"
	image := "127.0.0.1:15000/tftest-service:v1"
	ctx := context.Background()
	mounts := `
	mounts {
		source = docker_volume.foo.name
		target = "/mount/test"
		type   = "volume"
		read_only = true
		volume_options {
			labels {
				label = "env"
				value = "dev"
			}
			labels {
				label = "terraform"
				value = "true"
			}
		}
	}
	`
	hosts := `
	hosts {
		host = "testhost"
		ip = "10.0.1.0"
	}
	`
	logging := `
		name = "json-file"

		options = {
			max-size = "10m"
			max-file = "3"
		}
	`
	healthcheckInterval := "1s"
	healthcheckTimeout := "500ms"
	replicas := 2
	portsSpec := `
	ports {
		target_port    = "8080"
		published_port = "8081"
	}
	`

	// Step 2
	configData2 := "ewogICJwcmVmaXgiOiAiNTY3Igp9" // UPDATED to prefix: 567
	secretData2 := "ewogICJrZXkiOiAiUVdFUlRZIgp9" // UPDATED to YXCVB
	image2 := "127.0.0.1:15000/tftest-service:v2"
	healthcheckInterval2 := "2s"
	mounts2 := `
	mounts {
		source = docker_volume.foo.name
		target = "/mount/test"
		type   = "volume"
		read_only = true
		volume_options {
			labels {
				label = "env"
				value = "dev"
			}
			labels {
				label = "terraform"
				value = "true"
			}
		}
	}
	mounts {
		source = docker_volume.foo2.name
		target = "/mount/test2"
		type   = "volume"
		read_only = true
		volume_options {
			labels {
				label = "env"
				value = "dev"
			}
			labels {
				label = "terraform"
				value = "true"
			}
		}
	}
	`
	hosts2 := `
	hosts {
		host = "testhost2"
		ip = "10.0.2.2"
	}
	`
	logging2 := `
		name = "json-file"

		options = {
			max-size = "15m"
			max-file = "5"
		}
	`
	healthcheckTimeout2 := "800ms"
	replicas2 := 6
	portsSpec2 := `
	ports {
		target_port    = "8080"
		published_port = "8081"
	}
	ports {
		target_port    = "8080"
		published_port = "8082"
	}
	`

	// Step 3
	configData3 := configData2
	secretData3 := secretData2
	image3 := image2
	mounts3 := mounts2
	hosts3 := hosts2
	logging3 := logging2
	healthcheckInterval3 := healthcheckInterval2
	healthcheckTimeout3 := healthcheckTimeout2
	replicas3 := 3 // only decrease
	portsSpec3 := portsSpec2

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(updateMultiplePropertiesConfigConverge, configData, secretData, image, mounts, hosts, healthcheckInterval, healthcheckTimeout, logging, replicas, portsSpec),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-crihiadr"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1@sha256.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", strconv.Itoa(replicas)),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "3s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "continue"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "3s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.target_port", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.published_port", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dir", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dns_config.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.env.%", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.groups.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", healthcheckInterval),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", healthcheckTimeout),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hostname", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.1878413705.host", "testhost"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.1878413705.ip", "10.0.1.0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.isolation", "default"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.labels.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.privileges.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.stop_grace_period", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.user", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.name", "json-file"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-file", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-size", "10m"),
				),
			},
			{
				Config: fmt.Sprintf(updateMultiplePropertiesConfigConverge, configData2, secretData2, image2, mounts2, hosts2, healthcheckInterval2, healthcheckTimeout2, logging2, replicas2, portsSpec2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-crihiadr"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v2.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", strconv.Itoa(replicas2)),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "3s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "continue"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "3s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.target_port", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.published_port", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.1.target_port", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.1.published_port", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dir", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dns_config.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.env.%", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.groups.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", healthcheckInterval2),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", healthcheckTimeout2),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hostname", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.575059346.host", "testhost2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.575059346.ip", "10.0.2.2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.isolation", "default"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.labels.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.privileges.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.stop_grace_period", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.user", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.name", "json-file"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-file", "5"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-size", "15m"),
				),
			},
			{
				Config: fmt.Sprintf(updateMultiplePropertiesConfigConverge, configData3, secretData3, image3, mounts3, hosts3, healthcheckInterval3, healthcheckTimeout3, logging3, replicas3, portsSpec3),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-crihiadr"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v2.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", strconv.Itoa(replicas3)),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "3s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "continue"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "3s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.target_port", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.0.published_port", "8081"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.1.target_port", "8080"),
					resource.TestCheckResourceAttr("docker_service.foo", "endpoint_spec.0.ports.1.published_port", "8082"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dir", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dns_config.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.env.%", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.groups.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", healthcheckInterval3),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", healthcheckTimeout3),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hostname", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.575059346.host", "testhost2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.575059346.ip", "10.0.2.2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.isolation", "default"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.labels.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.#", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.privileges.#", "0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.stop_grace_period", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.user", ""),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.name", "json-file"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-file", "5"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-size", "15m"),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerService_nonExistingPrivateImageConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-privateimagedoesnotexist"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/idonoexist:latest"
						}
					}

					mode {
						replicated {
							replicas = 2
						}
					}

					converge_config {
						delay    = "7s"
						timeout  = "20s"
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.*did not converge after.*`),
				Check: resource.ComposeTestCheckFunc(
					isServiceRemoved("tftest-service-privateimagedoesnotexist"),
				),
			},
		},
	})
}

func TestAccDockerService_nonExistingPublicImageConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-publicimagedoesnotexist"
					task_spec {
						container_spec {
							image    = "stovogel/blablabla:part5"
						}
					}

					mode {
						replicated {
							replicas = 2
						}
					}

					converge_config {
						delay    = "7s"
						timeout  = "10s"
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.*did not converge after.*`),
				Check: resource.ComposeTestCheckFunc(
					isServiceRemoved("tftest-service-publicimagedoesnotexist"),
				),
			},
		},
	})
}

func TestAccDockerService_convergeAndStopGracefully(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				provider "docker" {
					registry_auth {
						address = "127.0.0.1:15000"
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-basic-converge"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
							healthcheck {
								test     = ["CMD", "curl", "-f", "localhost:8080/health"]
								interval = "5s"
								timeout  = "2s"
								start_period = "0s"
								retries  = 4
							}
						}
					}

					mode {
						replicated {
							replicas = 2
						}
					}

					endpoint_spec {
						ports {
							target_port = "8080"
						}
					}

					converge_config {
						delay    = "7s"
						timeout  = "3m"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic-converge"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					testValueHigherEqualThan("docker_service.foo", "endpoint_spec.0.ports.0.target_port", 8080),
					testValueHigherEqualThan("docker_service.foo", "endpoint_spec.0.ports.0.published_port", 30000),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerService_updateFailsAndRollbackConverge(t *testing.T) {
	image := "127.0.0.1:15000/tftest-service:v1"
	imageFail := "127.0.0.1:15000/tftest-service:v3"
	ctx := context.Background()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(updateFailsAndRollbackConvergeConfig, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-updateFailsAndRollbackConverge"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
				),
			},
			{
				Config:      fmt.Sprintf(updateFailsAndRollbackConvergeConfig, imageFail),
				ExpectError: regexp.MustCompile(`.*rollback completed.*`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-updateFailsAndRollbackConverge"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`127.0.0.1:15000/tftest-service:v1.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

const updateMultiplePropertiesConfigConverge = `
provider "docker" {
	alias = "private"
	registry_auth {
		address = "127.0.0.1:15000"
	}
}

resource "docker_volume" "foo" {
	name = "tftest-volume"
}

resource "docker_volume" "foo2" {
	name = "tftest-volume2"
}

resource "docker_config" "service_config" {
	name 			 = "tftest-myconfig-${uuid()}"
	data 			 = "%s"

	lifecycle {
		ignore_changes = ["name"]
		create_before_destroy = true
	}
}

resource "docker_secret" "service_secret" {
	name 			 = "tftest-tftest-mysecret-${replace(timestamp(),":", ".")}"
	data 			 = "%s"

	lifecycle {
		ignore_changes = ["name"]
		create_before_destroy = true
	}
}

resource "docker_service" "foo" {
	provider = "docker.private"
	name     = "tftest-fnf-service-up-crihiadr"

	task_spec {
		container_spec {
			image   = "%s"

			%s

			%s

			configs {
				config_id   = docker_config.service_config.id
				config_name = docker_config.service_config.name
				file_name   = "/configs.json"
			}

			secrets {
				secret_id   = docker_secret.service_secret.id
				secret_name = docker_secret.service_secret.name
				file_name   = "/secrets.json"
			}

			healthcheck {
				test     = ["CMD", "curl", "-f", "localhost:8080/health"]
				interval = "%s"
				timeout  = "%s"
				start_period = "1s"
				retries  = 2
			}

			stop_grace_period = "10s"
		}

		log_driver {
			%s
		}

	}

	mode {
		replicated {
			replicas = %d
		}
	}

	update_config {
		parallelism       = 2
		delay             = "3s"
		failure_action    = "continue"
		monitor           = "3s"
		max_failure_ratio = "0.5"
		order             = "start-first"
	}

	endpoint_spec {
		%s
	}

	converge_config {
		delay    = "7s"
		timeout  = "2m"
	}
}
`

const updateFailsAndRollbackConvergeConfig = `
provider "docker" {
	alias = "private"
	registry_auth {
		address = "127.0.0.1:15000"
	}
}

resource "docker_service" "foo" {
	provider = "docker.private"
	name     = "tftest-service-updateFailsAndRollbackConverge"
	task_spec {
		container_spec {
			image             = "%s"
			stop_grace_period = "10s"

			healthcheck {
				test     = ["CMD", "curl", "-f", "localhost:8080/health"]
				interval = "5s"
				timeout  = "2s"
				start_period = "0s"
				retries  = 4
			}
		}
	}

	mode {
		replicated {
			replicas = 2
		}
	}

	update_config {
		parallelism       = 1
		delay             = "5s"
		failure_action    = "rollback"
		monitor           = "10s"
		max_failure_ratio = "0.0"
		order             = "stop-first"
	}

	rollback_config {
		parallelism       = 1
		delay             = "1s"
		failure_action    = "pause"
		monitor           = "4s"
		max_failure_ratio = "0.0"
		order             = "stop-first"
	}

	endpoint_spec {
		mode = "vip"
		ports {
			name = "random"
			protocol     = "tcp"
			target_port 		 = "8080"
			published_port 		 = "8080"
			publish_mode = "ingress"
		}
	}

	converge_config {
		delay    = "7s"
		timeout  = "3m"
	}
}
`

// Helpers
// isServiceRemoved checks if a service was removed successfully
func isServiceRemoved(serviceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()
		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		filters := filters.NewArgs()
		filters.Add("name", serviceName)
		services, err := client.ServiceList(ctx, types.ServiceListOptions{
			Filters: filters,
		})
		if err != nil {
			return fmt.Errorf("Error listing service for name %s: %v", serviceName, err)
		}
		length := len(services)
		if length != 0 {
			return fmt.Errorf("Service should be removed but is running: %s", serviceName)
		}

		return nil
	}
}

// checkAndRemoveImages checks and removes all private images with
// the given pattern. This ensures that the image are not kept on the swarm nodes
// and the tests are independent of each other
func checkAndRemoveImages(ctx context.Context, s *terraform.State) error {
	retrySleepSeconds := 3
	maxRetryDeleteCount := 6
	imagePattern := "127.0.0.1:15000/tftest-service*"

	client := testAccProvider.Meta().(*ProviderConfig).DockerClient

	filters := filters.NewArgs()
	filters.Add("reference", imagePattern)
	images, err := client.ImageList(ctx, types.ImageListOptions{
		Filters: filters,
	})
	if err != nil {
		return err
	}

	retryDeleteCount := 0
	for i := 0; i < len(images); {
		image := images[i]
		_, err := client.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{
			Force: true,
		})
		if err != nil {
			if strings.Contains(err.Error(), "image is being used by running container") {
				if retryDeleteCount == maxRetryDeleteCount {
					return fmt.Errorf("could not delete image '%s' after %d retries", image.ID, maxRetryDeleteCount)
				}
				<-time.After(time.Duration(retrySleepSeconds) * time.Second)
				retryDeleteCount++
				continue
			}
			return err
		}
		i++
	}

	imagesAfterDelete, err := client.ImageList(ctx, types.ImageListOptions{
		Filters: filters,
	})
	if err != nil {
		return err
	}

	if len(imagesAfterDelete) != 0 {
		return fmt.Errorf("Expected images of pattern '%s' to be deleted, but there is/are still %d", imagePattern, len(imagesAfterDelete))
	}

	return nil
}
