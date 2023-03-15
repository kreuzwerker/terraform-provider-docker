package provider

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type byPortAndProtocol []string

func (s byPortAndProtocol) Len() int {
	return len(s)
}

func (s byPortAndProtocol) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byPortAndProtocol) Less(i, j int) bool {
	iSplit := strings.Split(string(s[i]), "/")
	iPort, _ := strconv.Atoi(iSplit[0])
	jSplit := strings.Split(string(s[j]), "/")
	jPort, _ := strconv.Atoi(jSplit[0])
	return iPort < jPort
}

func flattenContainerPorts(in nat.PortMap) []interface{} {
	out := make([]interface{}, 0)

	var internalPortKeys []string
	for portAndProtocolKeys := range in {
		internalPortKeys = append(internalPortKeys, string(portAndProtocolKeys))
	}
	sort.Sort(byPortAndProtocol(internalPortKeys))

	for _, portKey := range internalPortKeys {
		m := make(map[string]interface{})

		portBindings := in[nat.Port(portKey)]
		for _, portBinding := range portBindings {
			portProtocolSplit := strings.Split(string(portKey), "/")
			convertedInternal, _ := strconv.Atoi(portProtocolSplit[0])
			convertedExternal, _ := strconv.Atoi(portBinding.HostPort)
			m["internal"] = convertedInternal
			m["external"] = convertedExternal
			m["ip"] = portBinding.HostIP
			m["protocol"] = portProtocolSplit[1]
			out = append(out, m)
		}
	}
	return out
}

func flattenContainerNetworks(in *types.NetworkSettings) []interface{} {
	out := make([]interface{}, 0)
	if in == nil || in.Networks == nil || len(in.Networks) == 0 {
		return out
	}

	networks := in.Networks
	for networkName, networkData := range networks {
		m := make(map[string]interface{})
		m["network_name"] = networkName
		m["ip_address"] = networkData.IPAddress
		m["ip_prefix_length"] = networkData.IPPrefixLen
		m["gateway"] = networkData.Gateway
		m["global_ipv6_address"] = networkData.GlobalIPv6Address
		m["global_ipv6_prefix_length"] = networkData.GlobalIPv6PrefixLen
		m["ipv6_gateway"] = networkData.IPv6Gateway
		m["mac_address"] = networkData.MacAddress
		out = append(out, m)
	}
	return out
}

func stringListToStringSlice(stringList []interface{}) []string {
	ret := []string{}
	for _, v := range stringList {
		if v == nil {
			ret = append(ret, "")
			continue
		}
		ret = append(ret, v.(string))
	}
	return ret
}

func stringSetToStringSlice(stringSet *schema.Set) []string {
	ret := []string{}
	if stringSet == nil {
		return ret
	}
	for _, envVal := range stringSet.List() {
		ret = append(ret, envVal.(string))
	}
	return ret
}

func stringSetToMapStringString(stringSet *schema.Set) map[string]string {
	ret := map[string]string{}
	if stringSet == nil {
		return ret
	}
	for _, envVal := range stringSet.List() {
		envValSplit := strings.SplitN(envVal.(string), "=", 2)
		if len(envValSplit) != 2 {
			continue
		}
		ret[envValSplit[0]] = envValSplit[1]
	}
	return ret
}

func mapTypeMapValsToString(typeMap map[string]interface{}) map[string]string {
	mapped := make(map[string]string, len(typeMap))
	for k, v := range typeMap {
		mapped[k] = v.(string)
	}
	return mapped
}

// mapTypeMapValsToStringSlice maps a map to a slice with '=': e.g. foo = "bar" -> 'foo=bar'
func mapTypeMapValsToStringSlice(typeMap map[string]interface{}) []string {
	mapped := make([]string, 0)
	for k, v := range typeMap {
		if len(k) > 0 {
			mapped = append(mapped, k+"="+v.(string))
		}
	}
	return mapped
}

func portSetToDockerPorts(ports []interface{}) (map[nat.Port]struct{}, map[nat.Port][]nat.PortBinding) {
	retExposedPorts := map[nat.Port]struct{}{}
	retPortBindings := map[nat.Port][]nat.PortBinding{}

	for _, portInt := range ports {
		port := portInt.(map[string]interface{})
		internal := port["internal"].(int)
		protocol := port["protocol"].(string)

		exposedPort := nat.Port(strconv.Itoa(internal) + "/" + protocol)
		retExposedPorts[exposedPort] = struct{}{}

		portBinding := nat.PortBinding{}

		external, extOk := port["external"].(int)
		if extOk {
			portBinding.HostPort = strconv.Itoa(external)
		}

		ip, ipOk := port["ip"].(string)
		if ipOk {
			portBinding.HostIP = ip
		}

		if extOk || ipOk {
			retPortBindings[exposedPort] = append(retPortBindings[exposedPort], portBinding)
		}
	}

	return retExposedPorts, retPortBindings
}

func ulimitsToDockerUlimits(extraUlimits *schema.Set) []*units.Ulimit {
	retExtraUlimits := []*units.Ulimit{}

	for _, ulimitInt := range extraUlimits.List() {
		ulimits := ulimitInt.(map[string]interface{})
		u := &units.Ulimit{
			Name: ulimits["name"].(string),
			Soft: int64(ulimits["soft"].(int)),
			Hard: int64(ulimits["hard"].(int)),
		}
		retExtraUlimits = append(retExtraUlimits, u)
	}

	return retExtraUlimits
}

func extraHostsSetToContainerExtraHosts(extraHosts *schema.Set) []string {
	retExtraHosts := []string{}

	for _, hostInt := range extraHosts.List() {
		host := hostInt.(map[string]interface{})
		ip := host["ip"].(string)
		hostname := host["host"].(string)
		retExtraHosts = append(retExtraHosts, hostname+":"+ip)
	}

	return retExtraHosts
}

func volumeSetToDockerVolumes(volumes *schema.Set) (map[string]struct{}, []string, []string, error) {
	retVolumeMap := map[string]struct{}{}
	retHostConfigBinds := []string{}
	retVolumeFromContainers := []string{}

	for _, volumeInt := range volumes.List() {
		volume := volumeInt.(map[string]interface{})
		fromContainer := volume["from_container"].(string)
		containerPath := volume["container_path"].(string)
		volumeName := volume["volume_name"].(string)
		if len(volumeName) == 0 {
			volumeName = volume["host_path"].(string)
		}
		readOnly := volume["read_only"].(bool)

		switch {
		case len(fromContainer) == 0 && len(containerPath) == 0:
			return retVolumeMap, retHostConfigBinds, retVolumeFromContainers, errors.New("volume entry without container path or source container")
		case len(fromContainer) != 0 && len(containerPath) != 0:
			return retVolumeMap, retHostConfigBinds, retVolumeFromContainers, errors.New("both a container and a path specified in a volume entry")
		case len(fromContainer) != 0:
			retVolumeFromContainers = append(retVolumeFromContainers, fromContainer)
		case len(volumeName) != 0:
			readWrite := "rw"
			if readOnly {
				readWrite = "ro"
			}
			retVolumeMap[containerPath] = struct{}{}
			retHostConfigBinds = append(retHostConfigBinds, volumeName+":"+containerPath+":"+readWrite)
		default:
			retVolumeMap[containerPath] = struct{}{}
		}
	}

	return retVolumeMap, retHostConfigBinds, retVolumeFromContainers, nil
}

func deviceSetToDockerDevices(devices *schema.Set) []container.DeviceMapping {
	retDevices := []container.DeviceMapping{}
	for _, deviceInt := range devices.List() {
		deviceMap := deviceInt.(map[string]interface{})
		hostPath := deviceMap["host_path"].(string)
		containerPath := deviceMap["container_path"].(string)
		permissions := deviceMap["permissions"].(string)

		switch {
		case len(containerPath) == 0:
			containerPath = hostPath
			fallthrough
		case len(permissions) == 0:
			permissions = "rwm"
		}

		device := container.DeviceMapping{
			PathOnHost:        hostPath,
			PathInContainer:   containerPath,
			CgroupPermissions: permissions,
		}
		retDevices = append(retDevices, device)
	}
	return retDevices
}

func getDockerContainerMounts(container types.ContainerJSON) []map[string]interface{} {
	mounts := []map[string]interface{}{}
	for _, mount := range container.HostConfig.Mounts {
		m := map[string]interface{}{
			"target":    mount.Target,
			"source":    mount.Source,
			"type":      mount.Type,
			"read_only": mount.ReadOnly,
		}
		if mount.BindOptions != nil {
			m["bind_options"] = []map[string]interface{}{
				{
					"propagation": mount.BindOptions.Propagation,
				},
			}
		}
		if mount.VolumeOptions != nil {
			labels := []map[string]string{}
			for k, v := range mount.VolumeOptions.Labels {
				labels = append(labels, map[string]string{
					"label":  k,
					"volume": v,
				})
			}
			opt := map[string]interface{}{
				"no_copy": mount.VolumeOptions.NoCopy,
				"labels":  labels,
			}
			if mount.VolumeOptions.DriverConfig != nil {
				opt["driver_name"] = mount.VolumeOptions.DriverConfig.Name
				opt["driver_options"] = mount.VolumeOptions.DriverConfig.Options
			}
			m["volume_options"] = []map[string]interface{}{
				opt,
			}
		}
		if mount.TmpfsOptions != nil {
			m["tmpfs_options"] = []map[string]interface{}{
				{
					"size_bytes": mount.TmpfsOptions.SizeBytes,
					"mode":       mount.TmpfsOptions.Mode,
				},
			}
		}
		mounts = append(mounts, m)
	}

	return mounts
}

func flattenExtraHosts(in []string) []interface{} {
	extraHosts := make([]interface{}, len(in))
	for i, extraHost := range in {
		extraHostSplit := strings.Split(extraHost, ":")
		extraHosts[i] = map[string]interface{}{
			"host": extraHostSplit[0],
			"ip":   extraHostSplit[1],
		}
	}

	return extraHosts
}

func flattenUlimits(in []*units.Ulimit) []interface{} {
	ulimits := make([]interface{}, len(in))
	for i, ul := range in {
		ulimits[i] = map[string]interface{}{
			"name": ul.Name,
			"soft": ul.Soft,
			"hard": ul.Hard,
		}
	}

	return ulimits
}

func flattenDevices(in []container.DeviceMapping) []interface{} {
	devices := make([]interface{}, len(in))
	for i, device := range in {
		devices[i] = map[string]interface{}{
			"host_path":      device.PathOnHost,
			"container_path": device.PathInContainer,
			"permissions":    device.CgroupPermissions,
		}
	}

	return devices
}
