package docker

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/hashicorp/terraform/helper/schema"
)

var (
	creationTime time.Time
)

func resourceDockerContainerCreate(d *schema.ResourceData, meta interface{}) error {
	var err error
	client := meta.(*ProviderConfig).DockerClient

	var data Data
	if err := fetchLocalImages(&data, client); err != nil {
		return err
	}

	image := d.Get("image").(string)
	if _, ok := data.DockerImages[image]; !ok {
		if _, ok := data.DockerImages[image+":latest"]; !ok {
			return fmt.Errorf("Unable to find image %s", image)
		}
		image = image + ":latest"
	}

	config := &container.Config{
		Image:        image,
		Hostname:     d.Get("hostname").(string),
		Domainname:   d.Get("domainname").(string),
		AttachStdin:  d.Get("attach").(bool),
		AttachStdout: d.Get("attach").(bool),
		AttachStderr: d.Get("attach").(bool),
	}

	if v, ok := d.GetOk("env"); ok {
		config.Env = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("command"); ok {
		config.Cmd = stringListToStringSlice(v.([]interface{}))
		for _, v := range config.Cmd {
			if v == "" {
				return fmt.Errorf("values for command may not be empty")
			}
		}
	}

	if v, ok := d.GetOk("entrypoint"); ok {
		config.Entrypoint = stringListToStringSlice(v.([]interface{}))
	}

	if v, ok := d.GetOk("user"); ok {
		config.User = v.(string)
	}

	exposedPorts := map[nat.Port]struct{}{}
	portBindings := map[nat.Port][]nat.PortBinding{}

	if v, ok := d.GetOk("ports"); ok {
		exposedPorts, portBindings = portSetToDockerPorts(v.([]interface{}))
	}
	if len(exposedPorts) != 0 {
		config.ExposedPorts = exposedPorts
	}

	extraHosts := []string{}
	if v, ok := d.GetOk("host"); ok {
		extraHosts = extraHostsSetToDockerExtraHosts(v.(*schema.Set))
	}

	extraUlimits := []*units.Ulimit{}
	if v, ok := d.GetOk("ulimit"); ok {
		extraUlimits = ulimitsToDockerUlimits(v.(*schema.Set))
	}
	volumes := map[string]struct{}{}
	binds := []string{}
	volumesFrom := []string{}

	if v, ok := d.GetOk("volumes"); ok {
		volumes, binds, volumesFrom, err = volumeSetToDockerVolumes(v.(*schema.Set))
		if err != nil {
			return fmt.Errorf("Unable to parse volumes: %s", err)
		}
	}
	if len(volumes) != 0 {
		config.Volumes = volumes
	}

	if v, ok := d.GetOk("labels"); ok {
		config.Labels = mapTypeMapValsToString(v.(map[string]interface{}))
	}

	if value, ok := d.GetOk("healthcheck"); ok {
		config.Healthcheck = &container.HealthConfig{}
		if len(value.([]interface{})) > 0 {
			for _, rawHealthCheck := range value.([]interface{}) {
				rawHealthCheck := rawHealthCheck.(map[string]interface{})
				if testCommand, ok := rawHealthCheck["test"]; ok {
					config.Healthcheck.Test = stringListToStringSlice(testCommand.([]interface{}))
				}
				if rawInterval, ok := rawHealthCheck["interval"]; ok {
					config.Healthcheck.Interval, _ = time.ParseDuration(rawInterval.(string))
				}
				if rawTimeout, ok := rawHealthCheck["timeout"]; ok {
					config.Healthcheck.Timeout, _ = time.ParseDuration(rawTimeout.(string))
				}
				if rawStartPeriod, ok := rawHealthCheck["start_period"]; ok {
					config.Healthcheck.StartPeriod, _ = time.ParseDuration(rawStartPeriod.(string))
				}
				if rawRetries, ok := rawHealthCheck["retries"]; ok {
					config.Healthcheck.Retries, _ = rawRetries.(int)
				}
			}
		}
	}

	hostConfig := &container.HostConfig{
		Privileged:      d.Get("privileged").(bool),
		PublishAllPorts: d.Get("publish_all_ports").(bool),
		RestartPolicy: container.RestartPolicy{
			Name:              d.Get("restart").(string),
			MaximumRetryCount: d.Get("max_retry_count").(int),
		},
		AutoRemove: d.Get("rm").(bool),
		LogConfig: container.LogConfig{
			Type: d.Get("log_driver").(string),
		},
	}

	if len(portBindings) != 0 {
		hostConfig.PortBindings = portBindings
	}
	if len(extraHosts) != 0 {
		hostConfig.ExtraHosts = extraHosts
	}
	if len(binds) != 0 {
		hostConfig.Binds = binds
	}
	if len(volumesFrom) != 0 {
		hostConfig.VolumesFrom = volumesFrom
	}
	if len(extraUlimits) != 0 {
		hostConfig.Ulimits = extraUlimits
	}

	if v, ok := d.GetOk("capabilities"); ok {
		for _, capInt := range v.(*schema.Set).List() {
			capa := capInt.(map[string]interface{})
			hostConfig.CapAdd = stringSetToStringSlice(capa["add"].(*schema.Set))
			hostConfig.CapDrop = stringSetToStringSlice(capa["drop"].(*schema.Set))
			break
		}
	}

	if v, ok := d.GetOk("devices"); ok {
		hostConfig.Devices = deviceSetToDockerDevices(v.(*schema.Set))
	}

	if v, ok := d.GetOk("dns"); ok {
		hostConfig.DNS = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("dns_opts"); ok {
		hostConfig.DNSOptions = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("dns_search"); ok {
		hostConfig.DNSSearch = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("links"); ok {
		hostConfig.Links = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("memory"); ok {
		hostConfig.Memory = int64(v.(int)) * 1024 * 1024
	}

	if v, ok := d.GetOk("memory_swap"); ok {
		swap := int64(v.(int))
		if swap > 0 {
			swap = swap * 1024 * 1024
		}
		hostConfig.MemorySwap = swap
	}

	if v, ok := d.GetOk("cpu_shares"); ok {
		hostConfig.CPUShares = int64(v.(int))
	}

	if v, ok := d.GetOk("log_opts"); ok {
		hostConfig.LogConfig.Config = mapTypeMapValsToString(v.(map[string]interface{}))
	}

	networkingConfig := &network.NetworkingConfig{}
	if v, ok := d.GetOk("network_mode"); ok {
		hostConfig.NetworkMode = container.NetworkMode(v.(string))
	}

	if v, ok := d.GetOk("userns_mode"); ok {
		hostConfig.UsernsMode = container.UsernsMode(v.(string))
	}
	if v, ok := d.GetOk("pid_mode"); ok {
		hostConfig.PidMode = container.PidMode(v.(string))
	}

	var retContainer container.ContainerCreateCreatedBody

	if retContainer, err = client.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, d.Get("name").(string)); err != nil {
		return fmt.Errorf("Unable to create container: %s", err)
	}

	d.SetId(retContainer.ID)

	if v, ok := d.GetOk("networks"); ok {
		endpointConfig := &network.EndpointSettings{}
		if v, ok := d.GetOk("network_alias"); ok {
			endpointConfig.Aliases = stringSetToStringSlice(v.(*schema.Set))
		}

		if err := client.NetworkDisconnect(context.Background(), "bridge", retContainer.ID, false); err != nil {
			if !strings.Contains(err.Error(), "is not connected to the network bridge") {
				return fmt.Errorf("Unable to disconnect the default network: %s", err)
			}
		}

		for _, rawNetwork := range v.(*schema.Set).List() {
			networkID := rawNetwork.(string)
			if err := client.NetworkConnect(context.Background(), networkID, retContainer.ID, endpointConfig); err != nil {
				return fmt.Errorf("Unable to connect to network '%s': %s", networkID, err)
			}
		}
	}

	if v, ok := d.GetOk("upload"); ok {

		var mode int64
		for _, upload := range v.(*schema.Set).List() {
			content := upload.(map[string]interface{})["content"].(string)
			file := upload.(map[string]interface{})["file"].(string)
			executable := upload.(map[string]interface{})["executable"].(bool)

			buf := new(bytes.Buffer)
			tw := tar.NewWriter(buf)
			if executable {
				mode = 0744
			} else {
				mode = 0644
			}
			hdr := &tar.Header{
				Name: file,
				Mode: mode,
				Size: int64(len(content)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("Error creating tar archive: %s", err)
			}
			if _, err := tw.Write([]byte(content)); err != nil {
				return fmt.Errorf("Error creating tar archive: %s", err)
			}
			if err := tw.Close(); err != nil {
				return fmt.Errorf("Error creating tar archive: %s", err)
			}

			dstPath := "/"
			uploadContent := bytes.NewReader(buf.Bytes())
			options := types.CopyToContainerOptions{}
			if err := client.CopyToContainer(context.Background(), retContainer.ID, dstPath, uploadContent, options); err != nil {
				return fmt.Errorf("Unable to upload volume content: %s", err)
			}
		}
	}

	ctx := context.Background()
	creationTime = time.Now()
	if err := client.ContainerStart(ctx, retContainer.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("Unable to start container: %s", err)
	}

	if d.Get("attach").(bool) {
		statusCh, errCh := client.ContainerWait(ctx, retContainer.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("Unable to wait container end of execution: %s", err)
			}
		case <-statusCh:
		}
	}

	return resourceDockerContainerRead(d, meta)
}

func resourceDockerContainerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	apiContainer, err := fetchDockerContainer(d.Id(), client)
	if err != nil {
		return err
	}
	if apiContainer == nil {
		// This container doesn't exist anymore
		d.SetId("")
		return nil
	}

	var container types.ContainerJSON

	// TODO fix this with statefunc
	loops := 1 // if it hasn't just been created, don't delay
	if !creationTime.IsZero() {
		loops = 30 // with 500ms spacing, 15 seconds; ought to be plenty
	}
	sleepTime := 500 * time.Millisecond

	for i := loops; i > 0; i-- {
		container, err = client.ContainerInspect(context.Background(), apiContainer.ID)
		if err != nil {
			return fmt.Errorf("Error inspecting container %s: %s", apiContainer.ID, err)
		}

		jsonObj, _ := json.MarshalIndent(container, "", "\t")
		log.Printf("[INFO] Docker container inspect: %s", jsonObj)

		if container.State.Running ||
			!container.State.Running && !d.Get("must_run").(bool) {
			break
		}

		if creationTime.IsZero() { // We didn't just create it, so don't wait around
			return resourceDockerContainerDelete(d, meta)
		}

		finishTime, err := time.Parse(time.RFC3339, container.State.FinishedAt)
		if err != nil {
			return fmt.Errorf("Container finish time could not be parsed: %s", container.State.FinishedAt)
		}
		if finishTime.After(creationTime) {
			// It exited immediately, so error out so dependent containers
			// aren't started
			resourceDockerContainerDelete(d, meta)
			return fmt.Errorf("Container %s exited after creation, error was: %s", apiContainer.ID, container.State.Error)
		}

		time.Sleep(sleepTime)
	}

	// Handle the case of the for loop above running its course
	if !container.State.Running && d.Get("must_run").(bool) {
		resourceDockerContainerDelete(d, meta)
		return fmt.Errorf("Container %s failed to be in running state", apiContainer.ID)
	}

	if !container.State.Running {
		d.Set("exit_code", container.State.ExitCode)
	}

	// Read Network Settings
	if container.NetworkSettings != nil {
		// TODO remove deprecated attributes in next major
		d.Set("ip_address", container.NetworkSettings.IPAddress)
		d.Set("ip_prefix_length", container.NetworkSettings.IPPrefixLen)
		d.Set("gateway", container.NetworkSettings.Gateway)
		if container.NetworkSettings != nil && len(container.NetworkSettings.Networks) > 0 {
			// Still support deprecated outputs
			for _, settings := range container.NetworkSettings.Networks {
				d.Set("ip_address", settings.IPAddress)
				d.Set("ip_prefix_length", settings.IPPrefixLen)
				d.Set("gateway", settings.Gateway)
				break
			}
		}

		d.Set("bridge", container.NetworkSettings.Bridge)
		if err := d.Set("ports", flattenContainerPorts(container.NetworkSettings.Ports)); err != nil {
			log.Printf("[WARN] failed to set ports from API: %s", err)
		}
		if err := d.Set("network_data", flattenContainerNetworks(container.NetworkSettings)); err != nil {
			log.Printf("[WARN] failed to set network settings from API: %s", err)
		}
	}

	return nil
}

func resourceDockerContainerUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO call resourceDockerContainerRead here
	return nil
}

func resourceDockerContainerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	if d.Get("rm").(bool) {
		d.SetId("")
		return nil
	}

	if !d.Get("attach").(bool) {
		// Stop the container before removing if destroy_grace_seconds is defined
		if d.Get("destroy_grace_seconds").(int) > 0 {
			mapped := int32(d.Get("destroy_grace_seconds").(int))
			timeoutInSeconds := rand.Int31n(mapped)
			timeout := time.Duration(time.Duration(timeoutInSeconds) * time.Second)
			if err := client.ContainerStop(context.Background(), d.Id(), &timeout); err != nil {
				return fmt.Errorf("Error stopping container %s: %s", d.Id(), err)
			}
		}
	}

	removeOpts := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := client.ContainerRemove(context.Background(), d.Id(), removeOpts); err != nil {
		return fmt.Errorf("Error deleting container %s: %s", d.Id(), err)
	}

	waitOkC, errorC := client.ContainerWait(context.Background(), d.Id(), container.WaitConditionRemoved)
	select {
	case waitOk := <-waitOkC:
		log.Printf("[INFO] Container exited with code [%v]: '%s'", waitOk.StatusCode, d.Id())
	case err := <-errorC:
		if !(strings.Contains(err.Error(), "No such container") || strings.Contains(err.Error(), "is already in progress")) {
			return fmt.Errorf("Error waiting for container removal '%s': %s", d.Id(), err)
		}
	}

	d.SetId("")
	return nil
}

// TODO extract to structures_container.go
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
	var out = make([]interface{}, 0)

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
	var out = make([]interface{}, 0)
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
		out = append(out, m)
	}
	return out
}

// TODO move to separate flattener file
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

func fetchDockerContainer(ID string, client *client.Client) (*types.Container, error) {
	apiContainers, err := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})

	if err != nil {
		return nil, fmt.Errorf("Error fetching container information from Docker: %s\n", err)
	}

	for _, apiContainer := range apiContainers {
		if apiContainer.ID == ID {
			return &apiContainer, nil
		}
	}

	return nil, nil
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

		retPortBindings[exposedPort] = append(retPortBindings[exposedPort], portBinding)
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
func extraHostsSetToDockerExtraHosts(extraHosts *schema.Set) []string {
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
			return retVolumeMap, retHostConfigBinds, retVolumeFromContainers, errors.New("Volume entry without container path or source container")
		case len(fromContainer) != 0 && len(containerPath) != 0:
			return retVolumeMap, retHostConfigBinds, retVolumeFromContainers, errors.New("Both a container and a path specified in a volume entry")
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
