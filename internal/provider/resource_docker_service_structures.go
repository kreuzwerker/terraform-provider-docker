package provider

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ////////////
// flatteners
// flatten API objects to the terraform schema
// ////////////
// see https://learn.hashicorp.com/tutorials/terraform/provider-create?in=terraform/providers#add-flattening-functions
func flattenTaskSpec(in swarm.TaskSpec, d *schema.ResourceData) []interface{} {
	m := make(map[string]interface{})
	if in.ContainerSpec != nil {
		m["container_spec"] = flattenContainerSpec(in.ContainerSpec)
	}
	if in.Resources != nil {
		m["resources"] = flattenTaskResources(in.Resources)
	}
	if in.RestartPolicy != nil {
		m["restart_policy"] = flattenTaskRestartPolicy(in.RestartPolicy)
	}
	if in.Placement != nil {
		m["placement"] = flattenTaskPlacement(in.Placement)
	}
	m["force_update"] = in.ForceUpdate
	if len(in.Runtime) > 0 {
		m["runtime"] = in.Runtime
	}
	if len(in.Networks) > 0 {
		m["networks_advanced"] = flattenTaskNetworksAdvanced(in.Networks)
	}
	if in.LogDriver != nil {
		m["log_driver"] = flattenTaskLogDriver(in.LogDriver)
	}

	return []interface{}{m}
}

func flattenServiceMode(in swarm.ServiceMode) []interface{} {
	m := make(map[string]interface{})
	if in.Replicated != nil {
		m["replicated"] = flattenReplicated(in.Replicated)
	}
	if in.Global != nil {
		m["global"] = true
	} else {
		m["global"] = false
	}

	return []interface{}{m}
}

func flattenReplicated(in *swarm.ReplicatedService) []interface{} {
	out := make([]interface{}, 0)
	m := make(map[string]interface{})
	if in != nil {
		if in.Replicas != nil {
			replicas := int(*in.Replicas)
			m["replicas"] = replicas
		}
	}
	out = append(out, m)
	return out
}

func flattenServiceUpdateOrRollbackConfig(in *swarm.UpdateConfig) []interface{} {
	out := make([]interface{}, 0)
	if in == nil {
		return out
	}

	m := make(map[string]interface{})
	m["parallelism"] = in.Parallelism
	m["delay"] = shortDur(in.Delay)
	m["failure_action"] = in.FailureAction
	m["monitor"] = shortDur(in.Monitor)
	m["max_failure_ratio"] = strconv.FormatFloat(float64(in.MaxFailureRatio), 'f', 1, 64)
	m["order"] = in.Order
	out = append(out, m)
	return out
}

func flattenServiceEndpoint(in swarm.Endpoint) []interface{} {
	out := make([]interface{}, 0)
	m := make(map[string]interface{})
	m["mode"] = string(in.Spec.Mode)
	m["ports"] = flattenServicePorts(in.Ports)

	out = append(out, m)
	return out
}

func flattenServiceEndpointSpec(in *swarm.EndpointSpec) []interface{} {
	out := make([]interface{}, 0)
	m := make(map[string]interface{})
	m["mode"] = string(in.Mode)
	m["ports"] = flattenServicePorts(in.Ports)

	out = append(out, m)
	return out
}

// /// start TaskSpec
func flattenContainerSpec(in *swarm.ContainerSpec) []interface{} {
	out := make([]interface{}, 0)
	m := make(map[string]interface{})
	if len(in.Image) > 0 {
		m["image"] = in.Image
	}
	if len(in.Labels) > 0 {
		m["labels"] = mapToLabelSet(in.Labels)
	}
	if len(in.Command) > 0 {
		m["command"] = in.Command
	}
	if len(in.Args) > 0 {
		m["args"] = in.Args
	}
	if len(in.Hostname) > 0 {
		m["hostname"] = in.Hostname
	}
	if len(in.Env) > 0 {
		m["env"] = mapStringSliceToMap(in.Env)
	}
	if len(in.User) > 0 {
		m["user"] = in.User
	}
	if len(in.Dir) > 0 {
		m["dir"] = in.Dir
	}
	if len(in.Groups) > 0 {
		m["groups"] = in.Groups
	}
	if in.Privileges != nil {
		m["privileges"] = flattenPrivileges(in.Privileges)
	}
	if in.ReadOnly {
		m["read_only"] = in.ReadOnly
	}
	if len(in.Mounts) > 0 {
		m["mounts"] = flattenServiceMounts(in.Mounts)
	}
	if len(in.StopSignal) > 0 {
		m["stop_signal"] = in.StopSignal
	}
	if in.StopGracePeriod != nil {
		m["stop_grace_period"] = shortDur(*in.StopGracePeriod)
	}
	if in.Healthcheck != nil {
		m["healthcheck"] = flattenServiceHealthcheck(in.Healthcheck)
	}
	if len(in.Hosts) > 0 {
		m["hosts"] = flattenServiceHosts(in.Hosts)
	}
	if in.DNSConfig != nil {
		m["dns_config"] = flattenServiceDNSConfig(in.DNSConfig)
	}
	if len(in.Secrets) > 0 {
		m["secrets"] = flattenServiceSecrets(in.Secrets)
	}
	if len(in.Configs) > 0 {
		m["configs"] = flattenServiceConfigs(in.Configs)
	}
	if len(in.Isolation) > 0 {
		m["isolation"] = string(in.Isolation)
	}
	if len(in.Sysctls) > 0 {
		m["sysctl"] = in.Sysctls
	}
	out = append(out, m)
	return out
}

func flattenPrivileges(in *swarm.Privileges) []interface{} {
	if in == nil {
		return make([]interface{}, 0)
	}

	out := make([]interface{}, 1)
	m := make(map[string]interface{})

	if in.CredentialSpec != nil {
		credSpec := make([]interface{}, 1)
		internal := make(map[string]interface{})
		internal["file"] = in.CredentialSpec.File
		internal["registry"] = in.CredentialSpec.Registry
		credSpec[0] = internal
		m["credential_spec"] = credSpec
	}
	if in.SELinuxContext != nil {
		seLinuxContext := make([]interface{}, 1)
		internal := make(map[string]interface{})
		internal["disable"] = in.SELinuxContext.Disable
		internal["user"] = in.SELinuxContext.User
		internal["role"] = in.SELinuxContext.Role
		internal["type"] = in.SELinuxContext.Type
		internal["level"] = in.SELinuxContext.Level
		seLinuxContext[0] = internal
		m["se_linux_context"] = seLinuxContext
	}
	out[0] = m
	return out
}

func flattenServiceMounts(in []mount.Mount) *schema.Set {
	out := make([]interface{}, len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["target"] = v.Target
		m["source"] = v.Source
		m["type"] = string(v.Type)
		m["read_only"] = v.ReadOnly
		if v.BindOptions != nil {
			bindOptions := make([]interface{}, 0)
			bindOptionsItem := make(map[string]interface{})

			if len(v.BindOptions.Propagation) > 0 {
				bindOptionsItem["propagation"] = string(v.BindOptions.Propagation)
			}

			bindOptions = append(bindOptions, bindOptionsItem)
			m["bind_options"] = bindOptions
		}

		if v.VolumeOptions != nil {
			volumeOptions := make([]interface{}, 0)
			volumeOptionsItem := make(map[string]interface{})

			volumeOptionsItem["no_copy"] = v.VolumeOptions.NoCopy
			volumeOptionsItem["labels"] = mapToLabelSet(v.VolumeOptions.Labels)
			if v.VolumeOptions.DriverConfig != nil {
				if len(v.VolumeOptions.DriverConfig.Name) > 0 {
					volumeOptionsItem["driver_name"] = v.VolumeOptions.DriverConfig.Name
				}
				volumeOptionsItem["driver_options"] = mapStringStringToMapStringInterface(v.VolumeOptions.DriverConfig.Options)
			}

			volumeOptions = append(volumeOptions, volumeOptionsItem)
			m["volume_options"] = volumeOptions
		}

		if v.TmpfsOptions != nil {
			tmpfsOptions := make([]interface{}, 0)
			tmpfsOptionsItem := make(map[string]interface{})

			tmpfsOptionsItem["size_bytes"] = int(v.TmpfsOptions.SizeBytes)
			tmpfsOptionsItem["mode"] = v.TmpfsOptions.Mode.Perm

			tmpfsOptions = append(tmpfsOptions, tmpfsOptionsItem)
			m["tmpfs_options"] = tmpfsOptions
		}

		out[i] = m
	}
	taskSpecResource := resourceDockerService().Schema["task_spec"].Elem.(*schema.Resource)
	containerSpecResource := taskSpecResource.Schema["container_spec"].Elem.(*schema.Resource)
	mountsResource := containerSpecResource.Schema["mounts"].Elem.(*schema.Resource)
	f := schema.HashResource(mountsResource)
	return schema.NewSet(f, out)
}

func flattenServiceHealthcheck(in *container.HealthConfig) []interface{} {
	if in == nil {
		return make([]interface{}, 0)
	}

	out := make([]interface{}, 1)
	m := make(map[string]interface{})
	if len(in.Test) > 0 {
		m["test"] = in.Test
	}
	m["interval"] = shortDur(in.Interval)
	m["timeout"] = shortDur(in.Timeout)
	m["start_period"] = shortDur(in.StartPeriod)
	m["retries"] = in.Retries
	out[0] = m
	return out
}

func flattenServiceHosts(in []string) *schema.Set {
	out := make([]interface{}, len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		split := strings.Split(v, " ")
		log.Println("[DEBUG] got service hostnames to split:", split)
		m["ip"] = split[0]
		m["host"] = split[1]
		out[i] = m
	}
	taskSpecResource := resourceDockerService().Schema["task_spec"].Elem.(*schema.Resource)
	containerSpecResource := taskSpecResource.Schema["container_spec"].Elem.(*schema.Resource)
	hostsResource := containerSpecResource.Schema["hosts"].Elem.(*schema.Resource)
	f := schema.HashResource(hostsResource)
	return schema.NewSet(f, out)
}

func flattenServiceDNSConfig(in *swarm.DNSConfig) []interface{} {
	if in == nil {
		return make([]interface{}, 0)
	}

	out := make([]interface{}, 1)
	m := make(map[string]interface{})
	if len(in.Nameservers) > 0 {
		m["nameservers"] = in.Nameservers
	}
	if len(in.Search) > 0 {
		m["search"] = in.Search
	}
	if len(in.Options) > 0 {
		m["options"] = in.Options
	}
	out[0] = m
	return out
}

func flattenServiceSecrets(in []*swarm.SecretReference) *schema.Set {
	out := make([]interface{}, len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["secret_id"] = v.SecretID
		if len(v.SecretName) > 0 {
			m["secret_name"] = v.SecretName
		}
		if v.File != nil {
			m["file_name"] = v.File.Name
			if len(v.File.UID) > 0 {
				m["file_uid"] = v.File.UID
			}
			if len(v.File.GID) > 0 {
				m["file_gid"] = v.File.GID
			}
			m["file_mode"] = int(v.File.Mode)
		}
		out[i] = m
	}
	taskSpecResource := resourceDockerService().Schema["task_spec"].Elem.(*schema.Resource)
	containerSpecResource := taskSpecResource.Schema["container_spec"].Elem.(*schema.Resource)
	secretsResource := containerSpecResource.Schema["secrets"].Elem.(*schema.Resource)
	f := schema.HashResource(secretsResource)
	return schema.NewSet(f, out)
}

func flattenServiceConfigs(in []*swarm.ConfigReference) *schema.Set {
	out := make([]interface{}, len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["config_id"] = v.ConfigID
		if len(v.ConfigName) > 0 {
			m["config_name"] = v.ConfigName
		}
		if v.File != nil {
			m["file_name"] = v.File.Name
			if len(v.File.UID) > 0 {
				m["file_uid"] = v.File.UID
			}
			if len(v.File.GID) > 0 {
				m["file_gid"] = v.File.GID
			}
			m["file_mode"] = int(v.File.Mode)
		}
		out[i] = m
	}
	taskSpecResource := resourceDockerService().Schema["task_spec"].Elem.(*schema.Resource)
	containerSpecResource := taskSpecResource.Schema["container_spec"].Elem.(*schema.Resource)
	configsResource := containerSpecResource.Schema["configs"].Elem.(*schema.Resource)
	f := schema.HashResource(configsResource)
	return schema.NewSet(f, out)
}

func flattenTaskResources(in *swarm.ResourceRequirements) []interface{} {
	out := make([]interface{}, 0)
	if in != nil {
		m := make(map[string]interface{})
		m["limits"] = flattenResourceLimits(in.Limits)
		// TODO mvogel: name reservations
		m["reservation"] = flattenResourceReservations(in.Reservations)
		out = append(out, m)
	}
	return out
}

func flattenResourceLimits(in *swarm.Limit) []interface{} {
	out := make([]interface{}, 0)
	if in != nil {
		m := make(map[string]interface{})
		m["nano_cpus"] = in.NanoCPUs
		m["memory_bytes"] = in.MemoryBytes
		// TODO mavogel add pids
		// m["pids"] = in.Pids
		out = append(out, m)
	}
	return out
}

func flattenResourceReservations(in *swarm.Resources) []interface{} {
	out := make([]interface{}, 0)
	if in != nil {
		m := make(map[string]interface{})
		m["nano_cpus"] = in.NanoCPUs
		m["memory_bytes"] = in.MemoryBytes
		m["generic_resources"] = flattenResourceGenericResource(in.GenericResources)
		out = append(out, m)
	}
	return out
}

func flattenResourceGenericResource(in []swarm.GenericResource) []interface{} {
	out := make([]interface{}, 0)
	if len(in) > 0 {
		m := make(map[string]interface{})
		named := make([]string, 0)
		discrete := make([]string, 0)
		for _, genericResource := range in {
			if genericResource.NamedResourceSpec != nil {
				named = append(named, genericResource.NamedResourceSpec.Kind+"="+genericResource.NamedResourceSpec.Value)
			}
			if genericResource.DiscreteResourceSpec != nil {
				discrete = append(discrete, genericResource.DiscreteResourceSpec.Kind+"="+strconv.Itoa(int(genericResource.DiscreteResourceSpec.Value)))
			}
		}
		m["named_resources_spec"] = newStringSet(schema.HashString, named)
		m["discrete_resources_spec"] = newStringSet(schema.HashString, discrete)
		out = append(out, m)
	}
	return out
}

func flattenTaskRestartPolicy(in *swarm.RestartPolicy) []interface{} {
	if in == nil {
		return make([]interface{}, 0)
	}
	out := make([]interface{}, 1)
	m := make(map[string]interface{})
	if len(in.Condition) > 0 {
		m["condition"] = string(in.Condition)
	}
	if in.Delay != nil {
		m["delay"] = shortDur(*in.Delay)
	}
	if in.MaxAttempts != nil {
		mapped := *in.MaxAttempts
		m["max_attempts"] = int(mapped)
	}
	if in.Window != nil {
		m["window"] = shortDur(*in.Window)
	}
	out[0] = m
	return out
}

func flattenTaskPlacement(in *swarm.Placement) []interface{} {
	if in == nil {
		return make([]interface{}, 0)
	}
	out := make([]interface{}, 1)
	m := make(map[string]interface{})
	if len(in.Constraints) > 0 {
		m["constraints"] = newStringSet(schema.HashString, in.Constraints)
	}
	if len(in.Preferences) > 0 {
		m["prefs"] = flattenPlacementPrefs(in.Preferences)
	}
	if len(in.Platforms) > 0 {
		m["platforms"] = flattenPlacementPlatforms(in.Platforms)
	}
	m["max_replicas"] = in.MaxReplicas
	out[0] = m
	return out
}

func flattenPlacementPrefs(in []swarm.PlacementPreference) *schema.Set {
	if len(in) == 0 {
		return schema.NewSet(schema.HashString, make([]interface{}, 0))
	}

	out := make([]interface{}, len(in))
	for i, v := range in {
		out[i] = v.Spread.SpreadDescriptor
	}
	return schema.NewSet(schema.HashString, out)
}

func flattenPlacementPlatforms(in []swarm.Platform) *schema.Set {
	out := make([]interface{}, len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["architecture"] = v.Architecture
		m["os"] = v.OS
		out[i] = m
	}
	taskSpecResource := resourceDockerService().Schema["task_spec"].Elem.(*schema.Resource)
	placementResource := taskSpecResource.Schema["placement"].Elem.(*schema.Resource)
	f := schema.HashResource(placementResource)
	return schema.NewSet(f, out)
}

func flattenTaskNetworksAdvanced(in []swarm.NetworkAttachmentConfig) *schema.Set {
	out := make([]interface{}, len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["name"] = v.Target
		m["driver_opts"] = stringSliceToSchemaSet(mapTypeMapValsToStringSlice(mapStringStringToMapStringInterface(v.DriverOpts)))
		if len(v.Aliases) > 0 {
			m["aliases"] = stringSliceToSchemaSet(v.Aliases)
		}
		out[i] = m
	}
	taskSpecResource := resourceDockerService().Schema["task_spec"].Elem.(*schema.Resource)
	networksAdvancedResource := taskSpecResource.Schema["networks_advanced"].Elem.(*schema.Resource)
	f := schema.HashResource(networksAdvancedResource)
	return schema.NewSet(f, out)
}

func stringSliceToSchemaSet(in []string) *schema.Set {
	out := make([]interface{}, len(in))
	for i, v := range in {
		out[i] = v
	}
	return schema.NewSet(schema.HashString, out)
}

func flattenTaskLogDriver(in *swarm.Driver) []interface{} {
	if in == nil {
		return make([]interface{}, 0)
	}

	out := make([]interface{}, 1)
	m := make(map[string]interface{})
	m["name"] = in.Name
	if len(in.Options) > 0 {
		m["options"] = in.Options
	}
	out[0] = m
	return out
}

// /// end TaskSpec
// /// start EndpointSpec
func flattenServicePorts(in []swarm.PortConfig) []interface{} {
	out := make([]interface{}, len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["name"] = v.Name
		m["protocol"] = string(v.Protocol)
		m["target_port"] = int(v.TargetPort)
		m["published_port"] = int(v.PublishedPort)
		m["publish_mode"] = string(v.PublishMode)
		out[i] = m
	}
	return out
}

///// end EndpointSpec

// ////////////
// Mappers
// create API object from the terraform resource schema
// ////////////
// createServiceSpec creates the service spec: https://docs.docker.com/engine/api/v1.32/#operation/ServiceCreate
func createServiceSpec(d *schema.ResourceData) (swarm.ServiceSpec, error) {
	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: d.Get("name").(string),
		},
	}

	labels, err := createServiceLabels(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.Labels = labels

	taskTemplate, err := createServiceTaskSpec(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.TaskTemplate = taskTemplate

	mode, err := createServiceMode(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.Mode = mode

	updateConfig, err := createServiceUpdateConfig(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.UpdateConfig = updateConfig

	rollbackConfig, err := createServiceRollbackConfig(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.RollbackConfig = rollbackConfig

	endpointSpec, err := createServiceEndpointSpec(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.EndpointSpec = endpointSpec

	return serviceSpec, nil
}

// createServiceLabels creates the labels for the service
func createServiceLabels(d *schema.ResourceData) (map[string]string, error) {
	if v, ok := d.GetOk("labels"); ok {
		return labelSetToMap(v.(*schema.Set)), nil
	}
	return nil, nil
}

// == start taskSpec
// createServiceTaskSpec creates the task template for the service
func createServiceTaskSpec(d *schema.ResourceData) (swarm.TaskSpec, error) {
	taskSpec := swarm.TaskSpec{}
	if v, ok := d.GetOk("task_spec"); ok {
		if len(v.([]interface{})) > 0 {
			for _, rawTaskSpec := range v.([]interface{}) {
				rawTaskSpec := rawTaskSpec.(map[string]interface{})

				if rawContainerSpec, ok := rawTaskSpec["container_spec"]; ok {
					containerSpec, err := createContainerSpec(rawContainerSpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.ContainerSpec = containerSpec
				}

				if rawResourcesSpec, ok := rawTaskSpec["resources"]; ok {
					resources, err := createResources(rawResourcesSpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.Resources = resources
				}
				if rawRestartPolicySpec, ok := rawTaskSpec["restart_policy"]; ok {
					restartPolicy, err := createRestartPolicy(rawRestartPolicySpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.RestartPolicy = restartPolicy
				}
				if rawPlacementSpec, ok := rawTaskSpec["placement"]; ok {
					placement, err := createPlacement(rawPlacementSpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.Placement = placement
				}
				if rawForceUpdate, ok := rawTaskSpec["force_update"]; ok {
					taskSpec.ForceUpdate = uint64(rawForceUpdate.(int))
				}
				if rawRuntimeSpec, ok := rawTaskSpec["runtime"]; ok {
					taskSpec.Runtime = swarm.RuntimeType(rawRuntimeSpec.(string))
				}
				if rawNetworksSpec, ok := rawTaskSpec["networks_advanced"]; ok {
					networks, err := createServiceAdvancedNetworks(rawNetworksSpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.Networks = networks
				}
				if rawLogDriverSpec, ok := rawTaskSpec["log_driver"]; ok {
					logDriver, err := createLogDriver(rawLogDriverSpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.LogDriver = logDriver
				}
			}
		}
	}
	return taskSpec, nil
}

// createContainerSpec creates the container spec
func createContainerSpec(v interface{}) (*swarm.ContainerSpec, error) {
	containerSpec := swarm.ContainerSpec{}
	if len(v.([]interface{})) > 0 {
		for _, rawContainerSpec := range v.([]interface{}) {
			rawContainerSpec := rawContainerSpec.(map[string]interface{})
			if value, ok := rawContainerSpec["image"]; ok {
				containerSpec.Image = value.(string)
			}
			if value, ok := rawContainerSpec["labels"]; ok {
				containerSpec.Labels = labelSetToMap(value.(*schema.Set))
			}
			if value, ok := rawContainerSpec["command"]; ok {
				containerSpec.Command = stringListToStringSlice(value.([]interface{}))
			}
			if value, ok := rawContainerSpec["args"]; ok {
				containerSpec.Args = stringListToStringSlice(value.([]interface{}))
			}
			if value, ok := rawContainerSpec["hostname"]; ok {
				containerSpec.Hostname = value.(string)
			}
			if value, ok := rawContainerSpec["env"]; ok {
				containerSpec.Env = mapTypeMapValsToStringSlice(value.(map[string]interface{}))
			}
			if value, ok := rawContainerSpec["dir"]; ok {
				containerSpec.Dir = value.(string)
			}
			if value, ok := rawContainerSpec["user"]; ok {
				containerSpec.User = value.(string)
			}
			if value, ok := rawContainerSpec["groups"]; ok {
				containerSpec.Groups = stringListToStringSlice(value.([]interface{}))
			}
			if value, ok := rawContainerSpec["privileges"]; ok {
				if len(value.([]interface{})) > 0 {
					containerSpec.Privileges = &swarm.Privileges{}

					for _, rawPrivilegesSpec := range value.([]interface{}) {
						rawPrivilegesSpec := rawPrivilegesSpec.(map[string]interface{})

						if value, ok := rawPrivilegesSpec["credential_spec"]; ok {
							if len(value.([]interface{})) > 0 {
								containerSpec.Privileges.CredentialSpec = &swarm.CredentialSpec{}
								for _, rawCredentialSpec := range value.([]interface{}) {
									rawCredentialSpec := rawCredentialSpec.(map[string]interface{})
									if value, ok := rawCredentialSpec["file"]; ok {
										containerSpec.Privileges.CredentialSpec.File = value.(string)
									}
									if value, ok := rawCredentialSpec["registry"]; ok {
										containerSpec.Privileges.CredentialSpec.File = value.(string)
									}
								}
							}
						}
						if value, ok := rawPrivilegesSpec["se_linux_context"]; ok {
							if len(value.([]interface{})) > 0 {
								containerSpec.Privileges.SELinuxContext = &swarm.SELinuxContext{}
								for _, rawSELinuxContext := range value.([]interface{}) {
									rawSELinuxContext := rawSELinuxContext.(map[string]interface{})
									if value, ok := rawSELinuxContext["disable"]; ok {
										containerSpec.Privileges.SELinuxContext.Disable = value.(bool)
									}
									if value, ok := rawSELinuxContext["user"]; ok {
										containerSpec.Privileges.SELinuxContext.User = value.(string)
									}
									if value, ok := rawSELinuxContext["role"]; ok {
										containerSpec.Privileges.SELinuxContext.Role = value.(string)
									}
									if value, ok := rawSELinuxContext["type"]; ok {
										containerSpec.Privileges.SELinuxContext.Type = value.(string)
									}
									if value, ok := rawSELinuxContext["level"]; ok {
										containerSpec.Privileges.SELinuxContext.Level = value.(string)
									}
								}
							}
						}
					}
				}
			}
			if value, ok := rawContainerSpec["read_only"]; ok {
				containerSpec.ReadOnly = value.(bool)
			}
			if value, ok := rawContainerSpec["mounts"]; ok {
				mounts := []mount.Mount{}

				for _, rawMount := range value.(*schema.Set).List() {
					rawMount := rawMount.(map[string]interface{})
					mountType := mount.Type(rawMount["type"].(string))
					mountInstance := mount.Mount{
						Type:   mountType,
						Target: rawMount["target"].(string),
						Source: rawMount["source"].(string),
					}

					if value, ok := rawMount["read_only"]; ok {
						mountInstance.ReadOnly = value.(bool)
					}

					if value, ok := rawMount["bind_options"]; ok {
						// it always has 1 item (MaxItems = 1): the map: [map[propagation:]] even if the block is empty
						if len(value.([]interface{})) > 0 {
							mountInstance.BindOptions = &mount.BindOptions{}
							for _, rawBindOptions := range value.([]interface{}) {
								rawBindOptions := rawBindOptions.(map[string]interface{})
								if value, ok := rawBindOptions["propagation"]; ok {
									mountInstance.BindOptions.Propagation = mount.Propagation(value.(string))
								}
							}
						}
					}

					if value, ok := rawMount["volume_options"]; ok {
						if len(value.([]interface{})) > 0 {
							mountInstance.VolumeOptions = &mount.VolumeOptions{}
							for _, rawVolumeOptions := range value.([]interface{}) {
								rawVolumeOptions := rawVolumeOptions.(map[string]interface{})
								if value, ok := rawVolumeOptions["no_copy"]; ok {
									mountInstance.VolumeOptions.NoCopy = value.(bool)
								}
								if value, ok := rawVolumeOptions["labels"]; ok {
									mountInstance.VolumeOptions.Labels = labelSetToMap(value.(*schema.Set))
								}
								// because it is not possible to nest maps
								if value, ok := rawVolumeOptions["driver_name"]; ok {
									if mountInstance.VolumeOptions.DriverConfig == nil {
										mountInstance.VolumeOptions.DriverConfig = &mount.Driver{}
									}
									mountInstance.VolumeOptions.DriverConfig.Name = value.(string)
								}
								if value, ok := rawVolumeOptions["driver_options"]; ok {
									if mountInstance.VolumeOptions.DriverConfig == nil {
										mountInstance.VolumeOptions.DriverConfig = &mount.Driver{}
									}
									mountInstance.VolumeOptions.DriverConfig.Options = mapTypeMapValsToString(value.(map[string]interface{}))
								}
							}
						}
					}

					if value, ok := rawMount["tmpfs_options"]; ok {
						if len(value.([]interface{})) > 0 {
							mountInstance.TmpfsOptions = &mount.TmpfsOptions{}
							for _, rawTmpfsOptions := range value.([]interface{}) {
								rawTmpfsOptions := rawTmpfsOptions.(map[string]interface{})
								if value, ok := rawTmpfsOptions["size_bytes"]; ok {
									mountInstance.TmpfsOptions.SizeBytes = value.(int64)
								}
								if value, ok := rawTmpfsOptions["mode"]; ok {
									mountInstance.TmpfsOptions.Mode = os.FileMode(value.(int))
								}
							}
						}
					}

					mounts = append(mounts, mountInstance)
				}

				containerSpec.Mounts = mounts
			}
			if value, ok := rawContainerSpec["stop_signal"]; ok {
				containerSpec.StopSignal = value.(string)
			}
			if value, ok := rawContainerSpec["stop_grace_period"]; ok {
				parsed, _ := time.ParseDuration(value.(string))
				containerSpec.StopGracePeriod = &parsed
			}
			if value, ok := rawContainerSpec["healthcheck"]; ok {
				containerSpec.Healthcheck = &container.HealthConfig{}
				if len(value.([]interface{})) > 0 {
					for _, rawHealthCheck := range value.([]interface{}) {
						rawHealthCheck := rawHealthCheck.(map[string]interface{})
						if testCommand, ok := rawHealthCheck["test"]; ok {
							containerSpec.Healthcheck.Test = stringListToStringSlice(testCommand.([]interface{}))
						}
						if rawInterval, ok := rawHealthCheck["interval"]; ok {
							containerSpec.Healthcheck.Interval, _ = time.ParseDuration(rawInterval.(string))
						}
						if rawTimeout, ok := rawHealthCheck["timeout"]; ok {
							containerSpec.Healthcheck.Timeout, _ = time.ParseDuration(rawTimeout.(string))
						}
						if rawStartPeriod, ok := rawHealthCheck["start_period"]; ok {
							containerSpec.Healthcheck.StartPeriod, _ = time.ParseDuration(rawStartPeriod.(string))
						}
						if rawRetries, ok := rawHealthCheck["retries"]; ok {
							containerSpec.Healthcheck.Retries, _ = rawRetries.(int)
						}
					}
				}
			}
			if value, ok := rawContainerSpec["hosts"]; ok {
				containerSpec.Hosts = extraHostsSetToServiceExtraHosts(value.(*schema.Set))
			}
			if value, ok := rawContainerSpec["dns_config"]; ok {
				containerSpec.DNSConfig = &swarm.DNSConfig{}
				if len(v.([]interface{})) > 0 {
					for _, rawDNSConfig := range value.([]interface{}) {
						if rawDNSConfig != nil {
							rawDNSConfig := rawDNSConfig.(map[string]interface{})
							if nameservers, ok := rawDNSConfig["nameservers"]; ok {
								containerSpec.DNSConfig.Nameservers = stringListToStringSlice(nameservers.([]interface{}))
							}
							if search, ok := rawDNSConfig["search"]; ok {
								containerSpec.DNSConfig.Search = stringListToStringSlice(search.([]interface{}))
							}
							if options, ok := rawDNSConfig["options"]; ok {
								containerSpec.DNSConfig.Options = stringListToStringSlice(options.([]interface{}))
							}
						}
					}
				}
			}
			if value, ok := rawContainerSpec["secrets"]; ok {
				secrets := []*swarm.SecretReference{}

				for _, rawSecret := range value.(*schema.Set).List() {
					rawSecret := rawSecret.(map[string]interface{})
					rawFilemode := rawSecret["file_mode"].(int)
					secret := swarm.SecretReference{
						SecretID: rawSecret["secret_id"].(string),
						File: &swarm.SecretReferenceFileTarget{
							Name: rawSecret["file_name"].(string),
							UID:  rawSecret["file_uid"].(string),
							GID:  rawSecret["file_gid"].(string),
							Mode: os.FileMode(uint32(rawFilemode)),
						},
					}
					if value, ok := rawSecret["secret_name"]; ok {
						secret.SecretName = value.(string)
					}
					secrets = append(secrets, &secret)
				}
				containerSpec.Secrets = secrets
			}
			if value, ok := rawContainerSpec["configs"]; ok {
				configs := []*swarm.ConfigReference{}

				for _, rawConfig := range value.(*schema.Set).List() {
					rawConfig := rawConfig.(map[string]interface{})
					rawFilemode := rawConfig["file_mode"].(int)
					config := swarm.ConfigReference{
						ConfigID: rawConfig["config_id"].(string),
						File: &swarm.ConfigReferenceFileTarget{
							Name: rawConfig["file_name"].(string),
							UID:  rawConfig["file_uid"].(string),
							GID:  rawConfig["file_gid"].(string),
							Mode: os.FileMode(uint32(rawFilemode)),
						},
					}
					if value, ok := rawConfig["config_name"]; ok {
						config.ConfigName = value.(string)
					}
					configs = append(configs, &config)
				}
				containerSpec.Configs = configs
			}
			if value, ok := rawContainerSpec["isolation"]; ok {
				containerSpec.Isolation = container.Isolation(value.(string))
			}
			if value, ok := rawContainerSpec["sysctl"]; ok {
				containerSpec.Sysctls = mapTypeMapValsToString(value.(map[string]interface{}))
			}
		}
	}

	return &containerSpec, nil
}

// createResources creates the resource requirements for the service
func createResources(v interface{}) (*swarm.ResourceRequirements, error) {
	resources := swarm.ResourceRequirements{}
	if len(v.([]interface{})) > 0 {
		for _, rawResourcesSpec := range v.([]interface{}) {
			if rawResourcesSpec != nil {
				rawResourcesSpec := rawResourcesSpec.(map[string]interface{})
				if value, ok := rawResourcesSpec["limits"]; ok {
					if len(value.([]interface{})) > 0 {
						resources.Limits = &swarm.Limit{}
						for _, rawLimitsSpec := range value.([]interface{}) {
							rawLimitsSpec := rawLimitsSpec.(map[string]interface{})
							if value, ok := rawLimitsSpec["nano_cpus"]; ok {
								resources.Limits.NanoCPUs = int64(value.(int))
							}
							if value, ok := rawLimitsSpec["memory_bytes"]; ok {
								resources.Limits.MemoryBytes = int64(value.(int))
							}
						}
					}
				}
				if value, ok := rawResourcesSpec["reservation"]; ok {
					if len(value.([]interface{})) > 0 {
						resources.Reservations = &swarm.Resources{}
						for _, rawReservationSpec := range value.([]interface{}) {
							rawReservationSpec := rawReservationSpec.(map[string]interface{})
							if value, ok := rawReservationSpec["nano_cpus"]; ok {
								resources.Reservations.NanoCPUs = int64(value.(int))
							}
							if value, ok := rawReservationSpec["memory_bytes"]; ok {
								resources.Reservations.MemoryBytes = int64(value.(int))
							}
							if value, ok := rawReservationSpec["generic_resources"]; ok {
								resources.Reservations.GenericResources, _ = createGenericResources(value)
							}
						}
					}
				}
			}
		}
	}
	return &resources, nil
}

// createGenericResources creates generic resources for a container
func createGenericResources(value interface{}) ([]swarm.GenericResource, error) {
	genericResources := make([]swarm.GenericResource, 0)
	if len(value.([]interface{})) > 0 {
		for _, rawGenericResource := range value.([]interface{}) {
			rawGenericResource := rawGenericResource.(map[string]interface{})

			if rawNamedResources, ok := rawGenericResource["named_resources_spec"]; ok {
				for _, rawNamedResource := range rawNamedResources.(*schema.Set).List() {
					namedGenericResource := &swarm.NamedGenericResource{}
					splitted := strings.Split(rawNamedResource.(string), "=")
					namedGenericResource.Kind = splitted[0]
					namedGenericResource.Value = splitted[1]

					genericResource := swarm.GenericResource{}
					genericResource.NamedResourceSpec = namedGenericResource
					genericResources = append(genericResources, genericResource)
				}
			}
			if rawDiscreteResources, ok := rawGenericResource["discrete_resources_spec"]; ok {
				for _, rawDiscreteResource := range rawDiscreteResources.(*schema.Set).List() {
					discreteGenericResource := &swarm.DiscreteGenericResource{}
					splitted := strings.Split(rawDiscreteResource.(string), "=")
					discreteGenericResource.Kind = splitted[0]
					discreteGenericResource.Value, _ = strconv.ParseInt(splitted[1], 10, 64)

					genericResource := swarm.GenericResource{}
					genericResource.DiscreteResourceSpec = discreteGenericResource
					genericResources = append(genericResources, genericResource)
				}
			}
		}
	}
	return genericResources, nil
}

// createRestartPolicy creates the restart poliyc of the service
func createRestartPolicy(v interface{}) (*swarm.RestartPolicy, error) {
	restartPolicy := swarm.RestartPolicy{}
	rawRestartPolicySingleItem := v.([]interface{})
	if len(rawRestartPolicySingleItem) == 0 {
		return &restartPolicy, nil
	}
	// because it's a list with MaxItems=1
	rawRestartPolicy := rawRestartPolicySingleItem[0].(map[string]interface{})

	if v, ok := rawRestartPolicy["condition"]; ok {
		restartPolicy.Condition = swarm.RestartPolicyCondition(v.(string))
	}
	if v, ok := rawRestartPolicy["delay"]; ok {
		parsed, _ := time.ParseDuration(v.(string))
		restartPolicy.Delay = &parsed
	}
	if v, ok := rawRestartPolicy["max_attempts"]; ok {
		parsed := uint64(v.(int))
		restartPolicy.MaxAttempts = &parsed
	}
	if v, ok := rawRestartPolicy["window"]; ok {
		parsed, _ := time.ParseDuration(v.(string))
		restartPolicy.Window = &parsed
	}
	return &restartPolicy, nil
}

// createPlacement creates the placement strategy for the service
func createPlacement(v interface{}) (*swarm.Placement, error) {
	placement := swarm.Placement{}
	if len(v.([]interface{})) > 0 {
		for _, rawPlacement := range v.([]interface{}) {
			if rawPlacement != nil {
				rawPlacement := rawPlacement.(map[string]interface{})
				if v, ok := rawPlacement["constraints"]; ok {
					placement.Constraints = stringSetToStringSlice(v.(*schema.Set))
				}
				if v, ok := rawPlacement["prefs"]; ok {
					placement.Preferences = stringSetToPlacementPrefs(v.(*schema.Set))
				}
				if v, ok := rawPlacement["platforms"]; ok {
					placement.Platforms = mapSetToPlacementPlatforms(v.(*schema.Set))
				}
				if v, ok := rawPlacement["max_replicas"]; ok {
					placement.MaxReplicas = uint64(v.(int))
				}
			}
		}
	}

	return &placement, nil
}

// createServiceAdvancedNetworks creates the networks the service will be attachted to
func createServiceAdvancedNetworks(v interface{}) ([]swarm.NetworkAttachmentConfig, error) {
	networks := []swarm.NetworkAttachmentConfig{}
	if len(v.(*schema.Set).List()) > 0 {
		for _, rawNetwork := range v.(*schema.Set).List() {
			rawNetwork := rawNetwork.(map[string]interface{})
			networkID := rawNetwork["name"].(string)
			networkAliases := stringSetToStringSlice(rawNetwork["aliases"].(*schema.Set))
			network := swarm.NetworkAttachmentConfig{
				Target:  networkID,
				Aliases: networkAliases,
			}
			if driverOpts, ok := rawNetwork["driver_opts"]; ok {
				network.DriverOpts = stringSetToMapStringString(driverOpts.(*schema.Set))
			}
			networks = append(networks, network)
		}
	}
	return networks, nil
}

// createLogDriver creates the log driver for the service
func createLogDriver(v interface{}) (*swarm.Driver, error) {
	logDriver := swarm.Driver{}
	if len(v.([]interface{})) > 0 {
		for _, rawLogging := range v.([]interface{}) {
			rawLogging := rawLogging.(map[string]interface{})
			if rawName, ok := rawLogging["name"]; ok {
				logDriver.Name = rawName.(string)
			}
			if rawOptions, ok := rawLogging["options"]; ok {
				logDriver.Options = mapTypeMapValsToString(rawOptions.(map[string]interface{}))
			}
			// TODO SA4004: the surrounding loop is unconditionally terminated (staticcheck)
			return &logDriver, nil //nolint:staticcheck
		}
	}
	return nil, nil
}

// == end taskSpec

// createServiceMode creates the mode the service will run in
func createServiceMode(d *schema.ResourceData) (swarm.ServiceMode, error) {
	serviceMode := swarm.ServiceMode{}
	if v, ok := d.GetOk("mode"); ok {
		// because its a list
		if len(v.([]interface{})) > 0 {
			for _, rawMode := range v.([]interface{}) {
				// with a map
				rawMode := rawMode.(map[string]interface{})

				if rawReplicatedMode, replModeOk := rawMode["replicated"]; replModeOk {
					// with a list
					if len(rawReplicatedMode.([]interface{})) > 0 {
						for _, rawReplicatedModeInt := range rawReplicatedMode.([]interface{}) {
							// which is a map
							rawReplicatedModeMap := rawReplicatedModeInt.(map[string]interface{})
							log.Printf("[INFO] Setting service mode to 'replicated'")
							serviceMode.Replicated = &swarm.ReplicatedService{}
							if testReplicas, testReplicasOk := rawReplicatedModeMap["replicas"]; testReplicasOk {
								log.Printf("[INFO] Setting %v replicas", testReplicas)
								replicas := uint64(testReplicas.(int))
								serviceMode.Replicated.Replicas = &replicas
							}
						}
					}
				}
				if rawGlobalMode, globalModeOk := rawMode["global"]; globalModeOk && rawGlobalMode.(bool) {
					log.Printf("[INFO] Setting service mode to 'global' is %v", rawGlobalMode)
					serviceMode.Global = &swarm.GlobalService{}
				}
			}
		}
	}
	return serviceMode, nil
}

// createServiceUpdateConfig creates the service update config
func createServiceUpdateConfig(d *schema.ResourceData) (*swarm.UpdateConfig, error) {
	if v, ok := d.GetOk("update_config"); ok {
		return createUpdateOrRollbackConfig(v.([]interface{}))
	}
	return nil, nil
}

// createServiceRollbackConfig create the service rollback config
func createServiceRollbackConfig(d *schema.ResourceData) (*swarm.UpdateConfig, error) {
	if v, ok := d.GetOk("rollback_config"); ok {
		return createUpdateOrRollbackConfig(v.([]interface{}))
	}
	return nil, nil
}

// == start endpointSpec
// createServiceEndpointSpec creates the spec for the endpoint
func createServiceEndpointSpec(d *schema.ResourceData) (*swarm.EndpointSpec, error) {
	endpointSpec := swarm.EndpointSpec{}
	if v, ok := d.GetOk("endpoint_spec"); ok {
		if len(v.([]interface{})) > 0 {
			for _, rawEndpointSpec := range v.([]interface{}) {
				if rawEndpointSpec != nil {
					rawEndpointSpec := rawEndpointSpec.(map[string]interface{})
					if value, ok := rawEndpointSpec["mode"]; ok {
						endpointSpec.Mode = swarm.ResolutionMode(value.(string))
					}
					if value, ok := rawEndpointSpec["ports"]; ok {
						endpointSpec.Ports = portSetToServicePorts(value)
					}
				}
			}
		}
	}

	return &endpointSpec, nil
}

// portSetToServicePorts maps a set of ports to portConfig
func portSetToServicePorts(v interface{}) []swarm.PortConfig {
	retPortConfigs := []swarm.PortConfig{}
	if len(v.([]interface{})) > 0 {
		for _, portInt := range v.([]interface{}) {
			portConfig := swarm.PortConfig{}
			rawPort := portInt.(map[string]interface{})
			if value, ok := rawPort["name"]; ok {
				portConfig.Name = value.(string)
			}
			if value, ok := rawPort["protocol"]; ok {
				portConfig.Protocol = swarm.PortConfigProtocol(value.(string))
			}
			if value, ok := rawPort["target_port"]; ok {
				portConfig.TargetPort = uint32(value.(int))
			}
			if externalPort, ok := rawPort["published_port"]; ok {
				portConfig.PublishedPort = uint32(externalPort.(int))
			}
			if value, ok := rawPort["publish_mode"]; ok {
				portConfig.PublishMode = swarm.PortConfigPublishMode(value.(string))
			}

			retPortConfigs = append(retPortConfigs, portConfig)
		}
	}

	return retPortConfigs
}

// == end endpointSpec

// createUpdateOrRollbackConfig create the configuration for and update or rollback
func createUpdateOrRollbackConfig(config []interface{}) (*swarm.UpdateConfig, error) {
	updateConfig := swarm.UpdateConfig{}
	if len(config) > 0 {
		sc := config[0].(map[string]interface{})
		if v, ok := sc["parallelism"]; ok {
			updateConfig.Parallelism = uint64(v.(int))
		}
		if v, ok := sc["delay"]; ok {
			updateConfig.Delay, _ = time.ParseDuration(v.(string))
		}
		if v, ok := sc["failure_action"]; ok {
			updateConfig.FailureAction = v.(string)
		}
		if v, ok := sc["monitor"]; ok {
			updateConfig.Monitor, _ = time.ParseDuration(v.(string))
		}
		if v, ok := sc["max_failure_ratio"]; ok {
			value, _ := strconv.ParseFloat(v.(string), 64)
			updateConfig.MaxFailureRatio = float32(value)
		}
		if v, ok := sc["order"]; ok {
			updateConfig.Order = v.(string)
		}
	}

	return &updateConfig, nil
}

// createConvergeConfig creates the configuration for converging
func createConvergeConfig(config []interface{}) *convergeConfig {
	plainConvergeConfig := &convergeConfig{}
	if len(config) > 0 {
		for _, rawConvergeConfig := range config {
			rawConvergeConfig := rawConvergeConfig.(map[string]interface{})
			if delay, ok := rawConvergeConfig["delay"]; ok {
				plainConvergeConfig.delay, _ = time.ParseDuration(delay.(string))
			}
			if timeout, ok := rawConvergeConfig["timeout"]; ok {
				plainConvergeConfig.timeoutRaw, _ = timeout.(string)
				plainConvergeConfig.timeout, _ = time.ParseDuration(timeout.(string))
			}
		}
	}
	return plainConvergeConfig
}

// HELPERS
func shortDur(d time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}

func newStringSet(f schema.SchemaSetFunc, in []string) *schema.Set {
	out := make([]interface{}, len(in))
	for i, v := range in {
		out[i] = v
	}
	return schema.NewSet(f, out)
}

// mapStringSliceToMap maps a slice with '='  delimiter to as map: e.g.
// - 'foo=bar' -> foo = "bar"
// - 'foo=bar?p=baz' -> foo = "bar?p=baz"
func mapStringSliceToMap(in []string) map[string]string {
	mapped := make(map[string]string, len(in))
	for _, v := range in {
		if len(v) > 0 {
			splitted := strings.Split(v, "=")
			key := splitted[0]
			value := strings.Join(splitted[1:], "=")
			mapped[key] = value
		}
	}
	return mapped
}

// mapStringStringToMapStringInterface maps a string string map to a string interface map
func mapStringStringToMapStringInterface(in map[string]string) map[string]interface{} {
	if len(in) == 0 {
		return make(map[string]interface{})
	}

	mapped := make(map[string]interface{}, len(in))
	for k, v := range in {
		mapped[k] = v
	}
	return mapped
}

// stringSetToPlacementPrefs maps a string set to PlacementPreference
func stringSetToPlacementPrefs(stringSet *schema.Set) []swarm.PlacementPreference {
	ret := []swarm.PlacementPreference{}
	if stringSet == nil {
		return ret
	}
	for _, envVal := range stringSet.List() {
		ret = append(ret, swarm.PlacementPreference{
			Spread: &swarm.SpreadOver{
				SpreadDescriptor: envVal.(string),
			},
		})
	}
	return ret
}

// mapSetToPlacementPlatforms maps a string set to Platform
func mapSetToPlacementPlatforms(stringSet *schema.Set) []swarm.Platform {
	ret := []swarm.Platform{}
	if stringSet == nil {
		return ret
	}

	for _, rawPlatform := range stringSet.List() {
		rawPlatform := rawPlatform.(map[string]interface{})
		ret = append(ret, swarm.Platform{
			Architecture: rawPlatform["architecture"].(string),
			OS:           rawPlatform["os"].(string),
		})
	}

	return ret
}

func extraHostsSetToServiceExtraHosts(extraHosts *schema.Set) []string {
	retExtraHosts := []string{}

	for _, hostInt := range extraHosts.List() {
		host := hostInt.(map[string]interface{})
		ip := host["ip"].(string)
		hostname := host["host"].(string)
		// the delimiter is a 'space' + hostname and ip are switched
		// see https://github.com/kreuzwerker/terraform-provider-docker/issues/202#issuecomment-847715879
		retExtraHosts = append(retExtraHosts, ip+" "+hostname)
	}

	return retExtraHosts
}
