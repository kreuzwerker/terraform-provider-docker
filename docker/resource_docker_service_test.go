package docker

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// ----------------------------------------
// -----------    UNIT  TESTS   -----------
// ----------------------------------------

func TestDockerSecretFromRegistryAuth_basic(t *testing.T) {
	authConfigs := make(map[string]dc.AuthConfiguration)
	authConfigs["https://repo.my-company.com:8787"] = dc.AuthConfiguration{
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
	authConfigs := make(map[string]dc.AuthConfiguration)
	authConfigs["https://repo.my-company.com:8787"] = dc.AuthConfiguration{
		Username:      "myuser",
		Password:      "mypass",
		Email:         "",
		ServerAddress: "repo.my-company.com:8787",
	}
	authConfigs["https://nexus.my-fancy-company.com"] = dc.AuthConfiguration{
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

func checkAttribute(t *testing.T, name, actual, expected string) error {
	if actual != expected {
		t.Fatalf("bad authconfig attribute for '%q'\nExpected: %s\n     Got: %s", name, expected, actual)
	}

	return nil
}

// ----------------------------------------
// ----------- ACCEPTANCE TESTS -----------
// ----------------------------------------
// Fire and Forget
var serviceIDRegex = regexp.MustCompile(`[A-Za-z0-9_\+\.-]+`)

func TestAccDockerService_minimal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec {
							image = "127.0.0.1:15000/tftest-service:v1"
						}
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
				),
			},
		},
	})
}
func TestAccDockerService_full(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
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
					name = "tftest-service-basic"
				
					task_spec {
						container_spec {
							image = "127.0.0.1:15000/tftest-service:v1"
				
							labels {
								foo = "bar"
							}
				
							command  = ["ls"]
							args     = ["-las"]
							hostname = "my-fancy-service"
				
							env {
								MYFOO = "BAR"
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
				
							mounts = [
								{
									target      = "/mount/test"
									source      = "${docker_volume.test_volume.name}"
									type        = "volume"
									read_only   = true

									volume_options {
										no_copy = true
										labels {
											foo = "bar"
										}
										driver_name = "random-driver"
										driver_options {
											op1 = "val1"
										}
									}
								},
							]
				
							stop_signal       = "SIGTERM"
							stop_grace_period = "10s"
				
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
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
				
							secrets = [
								{
									secret_id   = "${docker_secret.service_secret.id}"
									secret_name = "${docker_secret.service_secret.name}"
									file_name = "/secrets.json"
								},
							]
				
							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name = "/configs.json"
								},
							]
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
						}
				
						force_update = 0
						runtime      = "container"
						networks     = ["${docker_network.test_network.id}"]
				
						log_driver {
							name = "json-file"
				
							options {
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
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.command.0", "ls"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.args.0", "-las"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hostname", "my-fancy-service"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.env.MYFOO", "BAR"),
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
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.816078185.target", "/mount/test"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.816078185.source", "tftest-volume"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.816078185.type", "volume"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.816078185.read_only", "true"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.816078185.volume_options.0.no_copy", "true"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.816078185.volume_options.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.816078185.volume_options.0.driver_name", "random-driver"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.816078185.volume_options.0.driver_options.op1", "val1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.stop_signal", "SIGTERM"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.stop_grace_period", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "5s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.1878413705.host", "testhost"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.1878413705.ip", "10.0.1.0"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dns_config.0.nameservers.0", "8.8.8.8"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dns_config.0.search.0", "example.org"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.dns_config.0.options.0", "timeout:3"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.resources.0.limits.0.nano_cpus", "1000000"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.resources.0.limits.0.memory_bytes", "536870912"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.restart_policy.condition", "on-failure"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.restart_policy.delay", "3s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.restart_policy.max_attempts", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.restart_policy.window", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.placement.0.constraints.4248571116", "node.role==manager"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.placement.0.prefs.1751004438", "spread=node.role.manager"),
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
				),
			},
		},
	})
}

func TestAccDockerService_partialReplicated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec = {
							image    = "127.0.0.1:15000/tftest-service:v1"
						}
					}
					mode {}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "1"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec = {
							image    = "127.0.0.1:15000/tftest-service:v1"
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
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "1"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec = {
							image    = "127.0.0.1:15000/tftest-service:v1"
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
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
				),
			},
		},
	})
}

func TestAccDockerService_basicGlobal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec = {
							image    = "127.0.0.1:15000/tftest-service:v1"
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
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.global", "true"),
				),
			},
		},
	})
}

func TestAccDockerService_GlobalAndReplicated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec = {
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
	})
}
func TestAccDockerService_GlobalWithConvergeConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic"
					task_spec {
						container_spec = {
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
	})
}

func TestAccDockerService_updateImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-fnf-service-up-image"
					task_spec {
						container_spec = {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"

							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 2
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
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
						}
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-image"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.stop_grace_period", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-fnf-service-up-image"
					task_spec {
						container_spec = {
							image    = "127.0.0.1:15000/tftest-service:v2"
							stop_grace_period = "10s"

							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 2
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
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
						}
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-image"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.stop_grace_period", "10s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
				),
			},
		},
	})
}

func TestAccDockerService_updateConfigReplicasImageAndHealthIncreaseAndDecreaseReplicas(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-fnf-service-up-crihiadr"

					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							
							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]
								
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								start_period = "0s"
								retries  = 2
							}
								
							stop_grace_period = "10s"
						}
					}

					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8081"
						}
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-fnf-service-up-crihiadr"

					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v2"

							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name = "/configs.json"
								},
							]

							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "2s"
								timeout  = "800ms"
								retries  = 4
							}

							stop_grace_period = "10s"
						}
					}

					mode {
						replicated {
							replicas = 6
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports = [
							{
								target_port    = "8080"
								published_port = "8081"
							},
							{
								target_port    = "8080"
								published_port = "8082"
							}
						] 
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "6"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-fnf-service-up-crihiadr"

					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v2"

							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name = "/configs.json"
								},
							]

							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "2s"
								timeout  = "800ms"
								retries  = 4
							}

							stop_grace_period = "10s"
						}
					}

					mode {
						replicated {
							replicas = 3
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports = [
							{
								target_port    = "8080"
								published_port = "8081"
							},
							{
								target_port    = "8080"
								published_port = "8082"
							}
						] 
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-fnf-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
		},
	})
}

// Converging tests
func TestAccDockerService_nonExistingPrivateImageConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-privateimagedoesnotexist"
					task_spec {
						container_spec = {
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
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-publicimagedoesnotexist"
					task_spec {
						container_spec = {
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

func TestAccDockerService_basicConvergeAndStopGracefully(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-basic-converge"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
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
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
				),
			},
		},
	})
}
func TestAccDockerService_updateFailsAndRollbackConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-updateFailsAndRollbackConverge"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
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
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-updateFailsAndRollbackConverge"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-updateFailsAndRollbackConverge"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v3"
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
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
				`,
				ExpectError: regexp.MustCompile(`.*rollback completed.*`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-updateFailsAndRollbackConverge"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
				),
			},
		},
	})
}

func TestAccDockerService_updateNetworksConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_network" "test_network" {
					name   = "tftest-network"
					driver = "overlay"
				}

				resource "docker_network" "test_network2" {
					name   = "tftest-network2"
					driver = "overlay"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-network"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
						}
						networks = ["${docker_network.test_network.id}"]
					}
					mode {
						replicated {
							replicas = 2
						}
					}
					
					
					endpoint_spec {
						mode = "vip"
					}
					converge_config {
						delay    = "7s"
						timeout  = "3m"
					}

				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-network"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.networks.#", "1"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_network" "test_network" {
					name   = "tftest-network"
					driver = "overlay"
				}

				resource "docker_network" "test_network2" {
					name   = "tftest-network2"
					driver = "overlay"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-network"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
						}
						networks = ["${docker_network.test_network2.id}"]
					}
					mode {
						replicated {
							replicas = 2
						}
					}

					endpoint_spec {
						mode = "vip"
					}

					converge_config {
						delay    = "7s"
						timeout  = "3m"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-network"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.networks.#", "1"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_network" "test_network" {
					name   = "tftest-network"
					driver = "overlay"
				}

				resource "docker_network" "test_network2" {
					name   = "tftest-network2"
					driver = "overlay"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-network"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
						}
						networks = [
							"${docker_network.test_network.id}",
							"${docker_network.test_network2.id}"
						]
					}

					mode {
						replicated {
							replicas = 2
						}
					}

					endpoint_spec {
						mode = "vip"
					}

					converge_config {
						delay    = "7s"
						timeout  = "3m"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-network"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.networks.#", "2"),
				),
			},
		},
	})
}
func TestAccDockerService_updateMountsConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_volume" "foo" {
					name = "tftest-volume"
				}

				resource "docker_volume" "foo2" {
					name = "tftest-volume2"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-mounts"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							mounts = [
								{
									source = "${docker_volume.foo.name}"
									target = "/mount/test"
									type   = "volume"
									read_only = true
									volume_options {
										labels {
											env = "dev"
											terraform = "true"
										}
									}
								}
							]
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
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-mounts"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.#", "1"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_volume" "foo" {
					name = "tftest-volume"
				}

				resource "docker_volume" "foo2" {
					name = "tftest-volume2"
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-mounts"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							mounts = [
								{
									source = "${docker_volume.foo.name}"
									target = "/mount/test"
									type   = "volume"
									read_only = true
									volume_options {
										labels {
											env = "dev"
											terraform = "true"
										}
									}
								},
								{
									source = "${docker_volume.foo2.name}"
									target = "/mount/test2"
									type   = "volume"
									read_only = true
									volume_options {
										labels {
											env = "dev"
											terraform = "true"
										}
									}
								}
							]
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
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-mounts"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.mounts.#", "2"),
				),
			},
		},
	})
}
func TestAccDockerService_updateHostsConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-hosts"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							hosts = [
								{
									host = "testhost"
									ip = "10.0.1.0"
								}
							]
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
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-hosts"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.#", "1"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-hosts"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							hosts = [
								{
									host = "testhost2"
									ip = "10.0.2.2"
								}
							]
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
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-hosts"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.#", "1"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-hosts"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							hosts = [
								{
									host = "testhost"
									ip = "10.0.1.0"
								},
								{
									host = "testhost2"
									ip = "10.0.2.2"
								}
							]
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
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-hosts"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.#", "2"),
				),
			},
		},
	})
}
func TestAccDockerService_updateLoggingConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-logging"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
						}

						log_driver {
							name = "json-file"
						
							options {
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

					converge_config {
						delay    = "7s"
						timeout  = "3m"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-logging"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.name", "json-file"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-size", "10m"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-file", "3"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-logging"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
						}
						log_driver {
							name = "json-file"
						
							options {
								max-size = "15m"
								max-file = "5"
							}
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
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-logging"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.name", "json-file"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.%", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-size", "15m"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.log_driver.0.options.max-file", "5"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-logging"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
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
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-logging"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
				),
			},
		},
	})
}

func TestAccDockerService_updateHealthcheckConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-healthcheck"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
		
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
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
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-healthcheck"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-healthcheck"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "2s"
								timeout  = "800ms"
								retries  = 2
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
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-healthcheck"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
				),
			},
		},
	})
}

func TestAccDockerService_updateIncreaseReplicasConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-increase-replicas"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"

							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 4
							}
						}
					}

					mode {
						replicated {
							replicas = 1
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-increase-replicas"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-increase-replicas"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
							
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 4
							}
						}
					}

					mode {
						replicated {
							replicas = 3
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-increase-replicas"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
		},
	})
}
func TestAccDockerService_updateDecreaseReplicasConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-decrease-replicas"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
							
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 4
							}
						}
					}

					mode {
						replicated {
							replicas = 5
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-decrease-replicas"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-decrease-replicas"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"

							stop_grace_period = "10s"
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 4
							}
						}
					}

					mode {
						replicated {
							replicas = 1
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-decrease-replicas"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
		},
	})
}

func TestAccDockerService_updateImageConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-image"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 2
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
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-image"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-image"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v2"
							stop_grace_period = "10s"
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 2
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
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.5"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-image"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
				),
			},
		},
	})
}

func TestAccDockerService_updateConfigConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 4
							}

							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]
						}
					}

					mode {
						replicated {
							replicas = 2
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.5"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
						}
					}
					
					converge_config {
						delay    = "7s"
						timeout  = "30s"
					}

				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 4
							}
							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]
						}
					}

					mode {
						replicated {
							replicas = 2
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
						}
					}
					
					converge_config {
						delay    = "7s"
						timeout  = "30s"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
		},
	})
}
func TestAccDockerService_updateConfigAndSecretConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_secret" "service_secret" {
					name 			 = "tftest-tftest-mysecret-${replace(timestamp(),":", ".")}"
					data 			 = "ewogICJrZXkiOiAiUVdFUlRZIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config-secret"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"

							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]

							secrets = [
								{
									secret_id   = "${docker_secret.service_secret.id}"
									secret_name = "${docker_secret.service_secret.name}"
									file_name   = "/secrets.json"
								},
							]
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 4
							}
							stop_grace_period = "10s"
						}
					}
					mode {
						replicated {
							replicas = 2
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config-secret"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name       = "tftest-myconfig-${uuid()}"
					data       = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_secret" "service_secret" {
					name       = "tftest-tftest-mysecret-${replace(timestamp(),":", ".")}"
					data       = "ewogICJrZXkiOiAiUVdFUlRZIgp9" # UPDATED to YXCVB

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config-secret"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"

							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]

							secrets = [
								{
									secret_id   = "${docker_secret.service_secret.id}"
									secret_name = "${docker_secret.service_secret.name}"
									file_name   = "/secrets.json"
								},
							]
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 4
							}
							stop_grace_period = "10s"
						}
					}
					mode {
						replicated {
							replicas = 2
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config-secret"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.configs.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.secrets.#", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
		},
	})
}
func TestAccDockerService_updatePortConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-port"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 2
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
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8081"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-port"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_service" "foo" {
					name     = "tftest-service-up-port"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							stop_grace_period = "10s"
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 2
							}
						}
					}

					mode {
						replicated {
							replicas = 4
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports = [
							{
								target_port    = "8080"
								published_port = "8081"
							},
							{
								target_port    = "8080"
								published_port = "8082"
							}
							] 
					}

					converge_config {
						delay    = "7s"
						timeout  = "3m"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-port"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
				),
			},
		},
	})
}
func TestAccDockerService_updateConfigReplicasImageAndHealthConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-crihc"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 2
							}
							stop_grace_period = "10s"
						}
					}

					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.5"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8081"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-crihc"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-crihc"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v2"
							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "2s"
								timeout  = "800ms"
								retries  = 4
							}
							stop_grace_period = "10s"
						}
					}

					mode {
						replicated {
							replicas = 4
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.5"
						order             = "start-first"
					}

					endpoint_spec {
						ports = [
							{
								target_port    = "8080"
								published_port = "8081"
							},
							{
								target_port    = "8080"
								published_port = "8082"
							}
						] 
					}

					converge_config {
						delay    = "7s"
						timeout  = "3m"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-crihc"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "4"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
		},
	})
}
func TestAccDockerService_updateConfigAndDecreaseReplicasConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config-dec-repl"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 4
							}
							stop_grace_period = "10s"
						}
					}

					mode {
						replicated {
							replicas = 5
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config-dec-repl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-config-dec-repl"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 4
							}
							stop_grace_period = "10s"
						}
					}

					mode {
						replicated {
							replicas = 1
						}
					}
					
					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.1"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8080"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-config-dec-repl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
		},
	})
}
func TestAccDockerService_updateConfigReplicasImageAndHealthIncreaseAndDecreaseReplicasConverge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiMTIzIgp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-crihiadr"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v1"
							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "1s"
								timeout  = "500ms"
								retries  = 2
							}
							stop_grace_period = "10s"
						}
					}

					mode {
						replicated {
							replicas = 2
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.5"
						order             = "start-first"
					}

					endpoint_spec {
						ports {
							target_port    = "8080"
							published_port = "8081"
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
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v1"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "500ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "2"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9" # UPDATED to prefix: 567

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-crihiadr"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v2"
							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "2s"
								timeout  = "800ms"
								retries  = 4
							}
							stop_grace_period = "10s"
						}
					}

					mode {
						replicated {
							replicas = 6
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.5"
						order             = "start-first"
					}

					endpoint_spec {
						ports = [
							{
								target_port    = "8080"
								published_port = "8081"
							},
							{
								target_port    = "8080"
								published_port = "8082"
							}
						] 
					}

					converge_config {
						delay    = "7s"
						timeout  = "3m"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "6"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
			resource.TestStep{
				Config: `
				resource "docker_config" "service_config" {
					name 			 = "tftest-myconfig-${uuid()}"
					data 			 = "ewogICJwcmVmaXgiOiAiNTY3Igp9"

					lifecycle {
						ignore_changes = ["name"]
						create_before_destroy = true
					}
				}

				resource "docker_service" "foo" {
					name     = "tftest-service-up-crihiadr"
					task_spec {
						container_spec {
							image    = "127.0.0.1:15000/tftest-service:v2"
							configs = [
								{
									config_id   = "${docker_config.service_config.id}"
									config_name = "${docker_config.service_config.name}"
									file_name   = "/configs.json"
								},
							]
							healthcheck {
								test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
								interval = "2s"
								timeout  = "800ms"
								retries  = 4
							}
							stop_grace_period = "10s"
						}
					}

					mode {
						replicated {
							replicas = 3
						}
					}

					update_config {
						parallelism       = 1
						delay             = "1s"
						failure_action    = "pause"
						monitor           = "1s"
						max_failure_ratio = "0.5"
						order             = "start-first"
					}

					endpoint_spec {
						ports = [
							{
								target_port    = "8080"
								published_port = "8081"
							},
							{
								target_port    = "8080"
								published_port = "8082"
							}
						] 
					}

					converge_config {
						delay    = "7s"
						timeout  = "3m"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-up-crihiadr"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", "127.0.0.1:15000/tftest-service:v2"),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "3"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.parallelism", "1"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.delay", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.failure_action", "pause"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.monitor", "1s"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.max_failure_ratio", "0.5"),
					resource.TestCheckResourceAttr("docker_service.foo", "update_config.0.order", "start-first"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.0", "CMD"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.1", "curl"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.2", "-f"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.test.3", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.interval", "2s"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.timeout", "800ms"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.healthcheck.0.retries", "4"),
				),
			},
		},
	})
}

func TestAccDockerService_privateConverge(t *testing.T) {
	registry := os.Getenv("DOCKER_REGISTRY_ADDRESS")
	image := os.Getenv("DOCKER_PRIVATE_IMAGE")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(`
					provider "docker" {
						alias = "private"
						registry_auth {
							address = "%s"
						}
					}

					resource "docker_service" "bar" {
						provider = "docker.private"
						name     = "tftest-service-bar"
						task_spec {
							container_spec {
								image    = "%s"
							}
						}
						mode {
							replicated {
								replicas = 2
							}
						}
					}
				`, registry, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.bar", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.bar", "name", "tftest-service-bar"),
					resource.TestCheckResourceAttr("docker_service.bar", "task_spec.0.container_spec.0.image", image),
				),
			},
		},
	})
}

// Helpers
func isServiceRemoved(serviceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		filter := make(map[string][]string)
		filter["name"] = []string{serviceName}
		services, err := client.ListServices(dc.ListServicesOptions{
			Filters: filter,
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
