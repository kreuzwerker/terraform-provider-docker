package docker

import (
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform/helper/schema"
)

func flattenTaskSpec(in swarm.TaskSpec) []interface{} {
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
	if in.ForceUpdate >= 0 {
		m["force_update"] = in.ForceUpdate
	}
	if len(in.Runtime) > 0 {
		m["runtime"] = in.Runtime
	}
	if len(in.Networks) > 0 {
		m["networks"] = flattenTaskNetworks(in.Networks)
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
	var out = make([]interface{}, 0, 0)
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
	var out = make([]interface{}, 0, 0)
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

func flattenServiceEndpointSpec(in swarm.EndpointSpec) []interface{} {
	var out = make([]interface{}, 0, 0)
	m := make(map[string]interface{})
	if len(in.Mode) > 0 {
		m["mode"] = in.Mode
	}
	if len(in.Ports) > 0 {
		m["ports"] = flattenServicePorts(in.Ports)
	}

	out = append(out, m)
	return out
}

///// start TaskSpec
func flattenContainerSpec(in *swarm.ContainerSpec) []interface{} {
	var out = make([]interface{}, 0, 0)
	m := make(map[string]interface{})
	if len(in.Image) > 0 {
		m["image"] = in.Image
	}
	if len(in.Labels) > 0 {
		m["labels"] = in.Labels
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
	out = append(out, m)
	return out
}

func flattenPrivileges(in *swarm.Privileges) []interface{} {
	if in == nil {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, 1, 1)
	m := make(map[string]interface{})

	if in.CredentialSpec != nil {
		credSpec := make([]interface{}, 1, 1)
		internal := make(map[string]interface{})
		internal["file"] = in.CredentialSpec.File
		internal["registry"] = in.CredentialSpec.Registry
		credSpec[0] = internal
		m["credential_spec"] = credSpec
	}
	if in.SELinuxContext != nil {
		seLinuxContext := make([]interface{}, 1, 1)
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
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["target"] = v.Target
		m["source"] = v.Source
		m["type"] = string(v.Type)
		m["read_only"] = v.ReadOnly
		if v.BindOptions != nil {
			bindOptions := make([]interface{}, 0, 0)
			bindOptionsItem := make(map[string]interface{}, 0)

			if len(v.BindOptions.Propagation) > 0 {
				bindOptionsItem["propagation"] = string(v.BindOptions.Propagation)
			}

			bindOptions = append(bindOptions, bindOptionsItem)
			m["bind_options"] = bindOptions
		}

		if v.VolumeOptions != nil {
			volumeOptions := make([]interface{}, 0, 0)
			volumeOptionsItem := make(map[string]interface{}, 0)

			volumeOptionsItem["no_copy"] = v.VolumeOptions.NoCopy
			volumeOptionsItem["labels"] = mapStringStringToMapStringInterface(v.VolumeOptions.Labels)
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
			tmpfsOptions := make([]interface{}, 0, 0)
			tmpfsOptionsItem := make(map[string]interface{}, 0)

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
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, 1, 1)
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
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		split := strings.Split(v, ":")
		m["host"] = split[0]
		m["ip"] = split[1]
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
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, 1, 1)
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
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["secret_id"] = v.SecretID
		if len(v.SecretName) > 0 {
			m["secret_name"] = v.SecretName
		}
		if v.File != nil {
			m["file_name"] = v.File.Name
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
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		m["config_id"] = v.ConfigID
		if len(v.ConfigName) > 0 {
			m["config_name"] = v.ConfigName
		}
		if v.File != nil {
			m["file_name"] = v.File.Name
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
	var out = make([]interface{}, 0, 0)
	if in != nil {
		m := make(map[string]interface{})
		m["limits"] = flattenResourceLimitsOrReservations(in.Limits)
		m["reservation"] = flattenResourceLimitsOrReservations(in.Reservations)
		out = append(out, m)
	}
	return out
}

func flattenResourceLimitsOrReservations(in *swarm.Resources) []interface{} {
	var out = make([]interface{}, 0, 0)
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
	var out = make([]interface{}, 0, 0)
	if in != nil && len(in) > 0 {
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

func flattenTaskRestartPolicy(in *swarm.RestartPolicy) map[string]interface{} {
	m := make(map[string]interface{})
	if len(in.Condition) > 0 {
		m["condition"] = string(in.Condition)
	}
	if in.Delay != nil {
		m["delay"] = shortDur(*in.Delay)
	}
	if in.MaxAttempts != nil {
		mapped := *in.MaxAttempts
		m["max_attempts"] = strconv.Itoa(int(mapped))
	}
	if in.Window != nil {
		m["window"] = shortDur(*in.Window)
	}
	return m
}

func flattenTaskPlacement(in *swarm.Placement) []interface{} {
	if in == nil {
		return make([]interface{}, 0, 0)
	}
	var out = make([]interface{}, 1, 1)
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
	out[0] = m
	return out
}

func flattenPlacementPrefs(in []swarm.PlacementPreference) *schema.Set {
	if in == nil || len(in) == 0 {
		return schema.NewSet(schema.HashString, make([]interface{}, 0, 0))
	}

	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = v.Spread.SpreadDescriptor
	}
	return schema.NewSet(schema.HashString, out)
}

func flattenPlacementPlatforms(in []swarm.Platform) *schema.Set {
	var out = make([]interface{}, len(in), len(in))
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

func flattenTaskNetworks(in []swarm.NetworkAttachmentConfig) *schema.Set {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = v.Target
	}
	return schema.NewSet(schema.HashString, out)
}

func flattenTaskLogDriver(in *swarm.Driver) []interface{} {
	if in == nil {
		return make([]interface{}, 0, 0)
	}

	var out = make([]interface{}, 1, 1)
	m := make(map[string]interface{})
	m["name"] = in.Name
	if len(in.Options) > 0 {
		m["options"] = in.Options
	}
	out[0] = m
	return out
}

///// end TaskSpec
///// start EndpointSpec
func flattenServicePorts(in []swarm.PortConfig) *schema.Set {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		m := make(map[string]interface{})
		if len(v.Name) > 0 {
			m["name"] = v.Name
		}
		m["protocol"] = string(v.Protocol)
		m["target_port"] = int(v.TargetPort)
		if v.PublishedPort > 0 {
			m["published_port"] = int(v.PublishedPort)
		}
		m["publish_mode"] = string(v.PublishMode)
		out[i] = m
	}
	endpointSpecResource := resourceDockerService().Schema["endpoint_spec"].Elem.(*schema.Resource)
	portsResource := endpointSpecResource.Schema["ports"].Elem.(*schema.Resource)
	f := schema.HashResource(portsResource)
	return schema.NewSet(f, out)
}

///// end EndpointSpec

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
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = v
	}
	return schema.NewSet(f, out)
}

// mapStringSliceToMap maps a slice with '='  delimiter to as map: e.g. 'foo=bar' -> foo = "bar"
func mapStringSliceToMap(in []string) map[string]string {
	mapped := make(map[string]string, len(in))
	for _, v := range in {
		if len(v) > 0 {
			splitted := strings.Split(v, "=")
			key := splitted[0]
			value := splitted[1]
			mapped[key] = value
		}
	}
	return mapped
}

// mapStringStringToMapStringInterface maps a string string map to a string interface map
func mapStringStringToMapStringInterface(in map[string]string) map[string]interface{} {
	if in == nil || len(in) == 0 {
		return make(map[string]interface{}, 0)
	}

	mapped := make(map[string]interface{}, len(in))
	for k, v := range in {
		mapped[k] = v
	}
	return mapped
}
