package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
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
				Config: loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServiceMinimalSpec"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
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
	var s swarm.Service

	// validates the inspected service json contains
	// all attributes set in the terraform spec so all mappers and flatteners
	// work as expected. This is to avoid bugs like
	// https://github.com/kreuzwerker/terraform-provider-docker/issues/202
	testCheckServiceInspect := func(*terraform.State) error {
		if len(s.Spec.Labels) != 1 || !mapEquals("servicelabel", "true", s.Spec.Labels) {
			return fmt.Errorf("Service Spec.Labels is wrong: %v", s.Spec.Labels)
		}

		if len(s.Spec.TaskTemplate.ContainerSpec.Command) != 1 ||
			s.Spec.TaskTemplate.ContainerSpec.Command[0] != "ls" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Command is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.Command)
		}

		if len(s.Spec.TaskTemplate.ContainerSpec.Args) != 1 ||
			s.Spec.TaskTemplate.ContainerSpec.Args[0] != "-las" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Args is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.Args)
		}

		if s.Spec.TaskTemplate.ContainerSpec.Hostname == "" ||
			s.Spec.TaskTemplate.ContainerSpec.Hostname != "my-fancy-service" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Hostname is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.Hostname)
		}

		// because the order is not deterministic
		if len(s.Spec.TaskTemplate.ContainerSpec.Env) != 2 ||
			(s.Spec.TaskTemplate.ContainerSpec.Env[0] != "URI=/api-call?param1=value1" && s.Spec.TaskTemplate.ContainerSpec.Env[0] != "MYFOO=BAR") ||
			(s.Spec.TaskTemplate.ContainerSpec.Env[1] != "URI=/api-call?param1=value1" && s.Spec.TaskTemplate.ContainerSpec.Env[1] != "MYFOO=BAR") {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Env is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.Env)
		}

		if s.Spec.TaskTemplate.ContainerSpec.Dir == "" ||
			s.Spec.TaskTemplate.ContainerSpec.Dir != "/root" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Dir is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.Dir)
		}

		if s.Spec.TaskTemplate.ContainerSpec.User == "" ||
			s.Spec.TaskTemplate.ContainerSpec.User != "root" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.User is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.User)
		}

		if len(s.Spec.TaskTemplate.ContainerSpec.Groups) != 2 ||
			s.Spec.TaskTemplate.ContainerSpec.Groups[0] != "docker" ||
			s.Spec.TaskTemplate.ContainerSpec.Groups[1] != "foogroup" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Groups is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.Groups)
		}

		if s.Spec.TaskTemplate.ContainerSpec.Privileges.CredentialSpec != nil {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Privileges.CredentialSpec is wrong: %v", s.Spec.TaskTemplate.ContainerSpec.Privileges.CredentialSpec)
		}

		if s.Spec.TaskTemplate.ContainerSpec.Privileges.SELinuxContext == nil ||
			s.Spec.TaskTemplate.ContainerSpec.Privileges.SELinuxContext.Disable != true ||
			s.Spec.TaskTemplate.ContainerSpec.Privileges.SELinuxContext.User != "user-label" ||
			s.Spec.TaskTemplate.ContainerSpec.Privileges.SELinuxContext.Role != "role-label" ||
			s.Spec.TaskTemplate.ContainerSpec.Privileges.SELinuxContext.Type != "type-label" ||
			s.Spec.TaskTemplate.ContainerSpec.Privileges.SELinuxContext.Level != "level-label" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Privileges.SELinuxContext is wrong: %v", s.Spec.TaskTemplate.ContainerSpec.Privileges.SELinuxContext)
		}

		if s.Spec.TaskTemplate.ContainerSpec.StopSignal == "" ||
			s.Spec.TaskTemplate.ContainerSpec.StopSignal != "SIGTERM" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.StopSignal is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.StopSignal)
		}

		if s.Spec.TaskTemplate.ContainerSpec.ReadOnly != true {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.ReadOnly is wrong: %v", s.Spec.TaskTemplate.ContainerSpec.ReadOnly)
		}

		if len(s.Spec.TaskTemplate.ContainerSpec.Mounts) != 1 ||
			s.Spec.TaskTemplate.ContainerSpec.Mounts[0].Type != "volume" ||
			s.Spec.TaskTemplate.ContainerSpec.Mounts[0].Source != "tftest-volume" ||
			s.Spec.TaskTemplate.ContainerSpec.Mounts[0].Target != "/mount/test" ||
			s.Spec.TaskTemplate.ContainerSpec.Mounts[0].ReadOnly != true ||
			s.Spec.TaskTemplate.ContainerSpec.Mounts[0].BindOptions != nil ||
			s.Spec.TaskTemplate.ContainerSpec.Mounts[0].Consistency != mount.Consistency("") ||
			s.Spec.TaskTemplate.ContainerSpec.Mounts[0].VolumeOptions.NoCopy != true ||
			!mapEquals("foo", "bar", s.Spec.TaskTemplate.ContainerSpec.Mounts[0].VolumeOptions.Labels) ||
			s.Spec.TaskTemplate.ContainerSpec.Mounts[0].VolumeOptions.DriverConfig.Name != "random-driver" ||
			!mapEquals("op1", "val1", s.Spec.TaskTemplate.ContainerSpec.Mounts[0].VolumeOptions.DriverConfig.Options) {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Mounts is wrong: %#v", s.Spec.TaskTemplate.ContainerSpec.Mounts)
		}

		if *s.Spec.TaskTemplate.ContainerSpec.StopGracePeriod != 10*time.Second {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.StopGracePeriod is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.StopGracePeriod)
		}

		if s.Spec.TaskTemplate.ContainerSpec.Healthcheck == nil ||
			len(s.Spec.TaskTemplate.ContainerSpec.Healthcheck.Test) != 4 ||
			s.Spec.TaskTemplate.ContainerSpec.Healthcheck.Test[0] != "CMD" ||
			s.Spec.TaskTemplate.ContainerSpec.Healthcheck.Test[1] != "curl" ||
			s.Spec.TaskTemplate.ContainerSpec.Healthcheck.Test[2] != "-f" ||
			s.Spec.TaskTemplate.ContainerSpec.Healthcheck.Test[3] != "localhost:8080/health" ||
			s.Spec.TaskTemplate.ContainerSpec.Healthcheck.Interval != 5*time.Second ||
			s.Spec.TaskTemplate.ContainerSpec.Healthcheck.Timeout != 2*time.Second ||
			time.Duration(s.Spec.TaskTemplate.ContainerSpec.Healthcheck.Retries) != 4 {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Healthcheck is wrong: %v", s.Spec.TaskTemplate.ContainerSpec.Healthcheck)
		}

		if len(s.Spec.TaskTemplate.ContainerSpec.Hosts) != 1 ||
			s.Spec.TaskTemplate.ContainerSpec.Hosts[0] != "10.0.1.0 testhost" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Hosts is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.Hosts)
		}

		if s.Spec.TaskTemplate.ContainerSpec.DNSConfig == nil ||
			len(s.Spec.TaskTemplate.ContainerSpec.DNSConfig.Nameservers) != 1 ||
			s.Spec.TaskTemplate.ContainerSpec.DNSConfig.Nameservers[0] != "8.8.8.8" ||
			len(s.Spec.TaskTemplate.ContainerSpec.DNSConfig.Search) != 1 ||
			s.Spec.TaskTemplate.ContainerSpec.DNSConfig.Search[0] != "example.org" ||
			len(s.Spec.TaskTemplate.ContainerSpec.DNSConfig.Options) != 1 ||
			s.Spec.TaskTemplate.ContainerSpec.DNSConfig.Options[0] != "timeout:3" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.DNSConfig is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.DNSConfig)
		}

		if len(s.Spec.TaskTemplate.ContainerSpec.Secrets) != 1 ||
			s.Spec.TaskTemplate.ContainerSpec.Secrets[0].SecretName != "tftest-mysecret" ||
			s.Spec.TaskTemplate.ContainerSpec.Secrets[0].File.Name != "/secrets.json" ||
			s.Spec.TaskTemplate.ContainerSpec.Secrets[0].File.UID != "0" ||
			s.Spec.TaskTemplate.ContainerSpec.Secrets[0].File.GID != "0" ||
			// nolint: staticcheck
			s.Spec.TaskTemplate.ContainerSpec.Secrets[0].File.Mode != os.FileMode(777) {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Secrets is wrong: %v", s.Spec.TaskTemplate.ContainerSpec.Secrets)
		}

		if len(s.Spec.TaskTemplate.ContainerSpec.Configs) != 1 ||
			s.Spec.TaskTemplate.ContainerSpec.Configs[0].ConfigName != "tftest-full-myconfig" ||
			s.Spec.TaskTemplate.ContainerSpec.Configs[0].File.Name != "/configs.json" ||
			s.Spec.TaskTemplate.ContainerSpec.Configs[0].File.UID != "0" ||
			s.Spec.TaskTemplate.ContainerSpec.Configs[0].File.GID != "0" ||
			s.Spec.TaskTemplate.ContainerSpec.Configs[0].File.Mode != os.FileMode(292) {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Configs is wrong: %v", s.Spec.TaskTemplate.ContainerSpec.Configs)
		}

		if s.Spec.TaskTemplate.ContainerSpec.Isolation == "" ||
			s.Spec.TaskTemplate.ContainerSpec.Isolation != "default" {
			return fmt.Errorf("Service Spec.TaskTemplate.ContainerSpec.Isolation is wrong: %s", s.Spec.TaskTemplate.ContainerSpec.Isolation)
		}

		if s.Spec.TaskTemplate.Resources == nil ||
			s.Spec.TaskTemplate.Resources.Limits == nil ||
			s.Spec.TaskTemplate.Resources.Limits.NanoCPUs != 1000000 ||
			s.Spec.TaskTemplate.Resources.Limits.MemoryBytes != 536870912 {
			return fmt.Errorf("Service Spec.TaskTemplate.Resources is wrong: %v", s.Spec.TaskTemplate.Resources)
		}

		if s.Spec.TaskTemplate.RestartPolicy == nil ||
			s.Spec.TaskTemplate.RestartPolicy.Condition != "on-failure" ||
			*s.Spec.TaskTemplate.RestartPolicy.Delay != 3*time.Second ||
			*s.Spec.TaskTemplate.RestartPolicy.MaxAttempts != 4 ||
			*s.Spec.TaskTemplate.RestartPolicy.Window != 10*time.Second {
			return fmt.Errorf("Service Spec.TaskTemplate.RestartPolicy is wrong: %v", s.Spec.TaskTemplate.RestartPolicy)
		}

		if s.Spec.TaskTemplate.Placement == nil ||
			len(s.Spec.TaskTemplate.Placement.Constraints) != 1 ||
			s.Spec.TaskTemplate.Placement.Constraints[0] != "node.role==manager" ||
			len(s.Spec.TaskTemplate.Placement.Preferences) != 1 ||
			s.Spec.TaskTemplate.Placement.Preferences[0].Spread == nil ||
			s.Spec.TaskTemplate.Placement.Preferences[0].Spread.SpreadDescriptor != "spread=node.role.manager" ||
			// s.Spec.TaskTemplate.Placement.MaxReplicas == uint64(2) || NOTE: mavogel: it's 0x2 in the log but does not work here either
			len(s.Spec.TaskTemplate.Placement.Platforms) != 1 ||
			s.Spec.TaskTemplate.Placement.Platforms[0].Architecture != "amd64" ||
			s.Spec.TaskTemplate.Placement.Platforms[0].OS != "linux" {
			return fmt.Errorf("Service Spec.TaskTemplate.Placement is wrong: %#v", s.Spec.TaskTemplate.Placement)
		}

		if s.Spec.TaskTemplate.Runtime == "" ||
			s.Spec.TaskTemplate.Runtime != "container" {
			return fmt.Errorf("Service Spec.TaskTemplate.Runtime is wrong: %s", s.Spec.TaskTemplate.Runtime)
		}

		if len(s.Spec.TaskTemplate.Networks) != 1 ||
			s.Spec.TaskTemplate.Networks[0].Target == "" {
			return fmt.Errorf("Service Spec.TaskTemplate.Networks is wrong: %s", s.Spec.TaskTemplate.Networks)
		}

		if s.Spec.TaskTemplate.LogDriver == nil ||
			s.Spec.TaskTemplate.LogDriver.Name != "json-file" ||
			!mapEquals("max-file", "3", s.Spec.TaskTemplate.LogDriver.Options) ||
			!mapEquals("max-size", "10m", s.Spec.TaskTemplate.LogDriver.Options) {
			return fmt.Errorf("Service Spec.TaskTemplate.LogDriver is wrong: %s", s.Spec.TaskTemplate.LogDriver)
		}

		if s.Spec.TaskTemplate.ForceUpdate != 0 {
			return fmt.Errorf("Service Spec.TaskTemplate.ForceUpdate is wrong: %v", s.Spec.TaskTemplate.ForceUpdate)
		}

		if s.Spec.Mode.Replicated == nil ||
			*s.Spec.Mode.Replicated.Replicas != uint64(2) {
			return fmt.Errorf("Service s.Spec.Mode.Replicated is wrong: %#v", s.Spec.Mode.Replicated)
		}

		if s.Spec.UpdateConfig == nil ||
			s.Spec.UpdateConfig.Parallelism != uint64(2) ||
			s.Spec.UpdateConfig.Delay != 10*time.Second ||
			s.Spec.UpdateConfig.FailureAction != "pause" ||
			s.Spec.UpdateConfig.Monitor != 5*time.Second ||
			s.Spec.UpdateConfig.MaxFailureRatio != 0.1 ||
			s.Spec.UpdateConfig.Order != "start-first" {
			return fmt.Errorf("Service s.Spec.UpdateConfig is wrong: %#v", s.Spec.UpdateConfig)
		}

		if s.Spec.RollbackConfig == nil ||
			s.Spec.RollbackConfig.Parallelism != uint64(2) ||
			s.Spec.RollbackConfig.Delay != 5*time.Millisecond ||
			s.Spec.RollbackConfig.FailureAction != "pause" ||
			s.Spec.RollbackConfig.Monitor != 10*time.Hour ||
			s.Spec.RollbackConfig.MaxFailureRatio != 0.9 ||
			s.Spec.RollbackConfig.Order != "stop-first" {
			return fmt.Errorf("Service s.Spec.RollbackConfig is wrong: %#v", s.Spec.RollbackConfig)
		}

		if s.Spec.EndpointSpec == nil ||
			s.Spec.EndpointSpec.Mode != swarm.ResolutionModeVIP ||
			len(s.Spec.EndpointSpec.Ports) != 1 ||
			s.Spec.EndpointSpec.Ports[0].Name != "random" ||
			s.Spec.EndpointSpec.Ports[0].Protocol != swarm.PortConfigProtocolTCP ||
			s.Spec.EndpointSpec.Ports[0].TargetPort != uint32(8080) ||
			s.Spec.EndpointSpec.Ports[0].PublishedPort != uint32(8080) ||
			s.Spec.EndpointSpec.Ports[0].PublishMode != swarm.PortConfigPublishModeIngress {
			return fmt.Errorf("Service s.Spec.EndpointSpec is wrong: %#v", s.Spec.EndpointSpec)
		}

		if s.Endpoint.Spec.Mode != swarm.ResolutionModeVIP ||
			len(s.Endpoint.Spec.Ports) != 1 ||
			s.Endpoint.Spec.Ports[0].Name != "random" ||
			s.Endpoint.Spec.Ports[0].Protocol != swarm.PortConfigProtocolTCP ||
			s.Endpoint.Spec.Ports[0].TargetPort != uint32(8080) ||
			s.Endpoint.Spec.Ports[0].PublishedPort != uint32(8080) ||
			s.Endpoint.Spec.Ports[0].PublishMode != swarm.PortConfigPublishModeIngress ||
			len(s.Endpoint.Ports) != 1 ||
			s.Endpoint.Ports[0].Name != "random" ||
			s.Endpoint.Ports[0].Protocol != swarm.PortConfigProtocolTCP ||
			s.Endpoint.Ports[0].TargetPort != uint32(8080) ||
			s.Endpoint.Ports[0].PublishedPort != uint32(8080) ||
			s.Endpoint.Ports[0].PublishMode != swarm.PortConfigPublishModeIngress ||
			len(s.Endpoint.VirtualIPs) != 2 {
			return fmt.Errorf("Service s.Endpoint is wrong: %#v", s.Endpoint)
		}

		return nil
	}

	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServiceFullSpec"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					testCheckLabelMap("docker_service.foo", "labels", map[string]string{"servicelabel": "true"}),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
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
					testAccServiceRunning("docker_service.foo", &s),
					testCheckServiceInspect,
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
				Config: loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServicePartialReplicationConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "1"),
				),
			},
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServicePartialReplicationConfigStep2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "1"),
				),
			},
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServicePartialReplicationConfigStep3"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
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
				Config: loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServiceGlobalReplicationMode"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256@sha256.*`)),
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
				Config:      loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServiceConflictingGlobalAndReplicated"),
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
				Config:      loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServiceConflictingGlobalModeAndConverge"),
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
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServicePrivateImageConverge"), registry, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-foo"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
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
				Config:      loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServiceNonExistingPrivateImageConverge"),
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
				Config:      loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServicenonExistingPublicImageConverge"),
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
				Config: loadTestConfiguration(t, RESOURCE, "docker_service", "testAccDockerServiceConvergeAndStopGracefully"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-basic-converge"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
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
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_service", "updateFailsAndRollbackConvergeConfig"), image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-updateFailsAndRollbackConverge"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
				),
			},
			{
				Config:      fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_service", "updateFailsAndRollbackConvergeConfig"), imageFail),
				ExpectError: regexp.MustCompile(`.*rollback completed.*`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_service.foo", "id", serviceIDRegex),
					resource.TestCheckResourceAttr("docker_service.foo", "name", "tftest-service-updateFailsAndRollbackConverge"),
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
					resource.TestCheckResourceAttr("docker_service.foo", "mode.0.replicated.0.replicas", "2"),
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
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
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
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.0.host", "testhost"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.0.ip", "10.0.1.0"),
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
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
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
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.0.host", "testhost2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.0.ip", "10.0.2.2"),
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
					resource.TestMatchResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.image", regexp.MustCompile(`sha256.*`)),
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
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.0.host", "testhost2"),
					resource.TestCheckResourceAttr("docker_service.foo", "task_spec.0.container_spec.0.hosts.0.ip", "10.0.2.2"),
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
	name = "tftest-myconfig-${uuid()}"
	data = "%s"
  
	lifecycle {
	  ignore_changes        = ["name"]
	  create_before_destroy = true
	}
  }
  
  resource "docker_secret" "service_secret" {
	name = "tftest-tftest-mysecret-${replace(timestamp(), ":", ".")}"
	data = "%s"
  
	lifecycle {
	  ignore_changes        = ["name"]
	  create_before_destroy = true
	}
  }

  data "docker_registry_image" "tftest_image" {
	provider             = "docker.private"
	name                 = "%s"
	insecure_skip_verify = true
  }
  resource "docker_image" "tftest_image" {
	provider      = "docker.private"
	name          = data.docker_registry_image.tftest_image.name
	keep_locally  = true
	pull_triggers = [data.docker_registry_image.tftest_image.sha256_digest]
  }
  
  resource "docker_service" "foo" {
	provider = "docker.private"
	name     = "tftest-fnf-service-up-crihiadr"
  
	task_spec {
	  container_spec {
		image = docker_image.tftest_image.latest
  
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
		  test         = ["CMD", "curl", "-f", "localhost:8080/health"]
		  interval     = "%s"
		  timeout      = "%s"
		  start_period = "1s"
		  retries      = 2
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
	  delay   = "7s"
	  timeout = "2m"
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

func testAccServiceRunning(resourceName string, service *swarm.Service) resource.TestCheckFunc {
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
		inspectedService, _, err := client.ServiceInspectWithRaw(ctx, rs.Primary.ID, types.ServiceInspectOptions{})
		if err != nil {
			return fmt.Errorf("Service with ID '%s': %w", rs.Primary.ID, err)
		}

		// we set the value to the pointer to be able to use the value
		// outside of the function
		*service = inspectedService
		return nil

	}
}
