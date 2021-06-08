package provider

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	containerReadRefreshTimeout             = 15 * time.Second
	containerReadRefreshWaitBeforeRefreshes = 100 * time.Millisecond
	containerReadRefreshDelay               = 100 * time.Millisecond
)

var (
	errContainerFailedToBeCreated        = errors.New("container failed to be created")
	errContainerFailedToBeDeleted        = errors.New("container failed to be deleted")
	errContainerExitedImmediately        = errors.New("container exited immediately")
	errContainerFailedToBeInRunningState = errors.New("container failed to be in running state")
)

// NOTE mavogel: we keep this global var for tracking
// the time in the create and read func
var creationTime time.Time

func resourceDockerContainerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var err error
	client := meta.(*ProviderConfig).DockerClient
	authConfigs := meta.(*ProviderConfig).AuthConfigs
	image := d.Get("image").(string)
	_, err = findImage(ctx, image, client, authConfigs)
	if err != nil {
		return diag.Errorf("Unable to create container with image %s: %s", image, err)
	}

	config := &container.Config{
		Image:      image,
		Hostname:   d.Get("hostname").(string),
		Domainname: d.Get("domainname").(string),
		Tty:        d.Get("tty").(bool),
		OpenStdin:  d.Get("stdin_open").(bool),
	}

	if v, ok := d.GetOk("env"); ok {
		config.Env = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("command"); ok {
		config.Cmd = stringListToStringSlice(v.([]interface{}))
		for _, v := range config.Cmd {
			if v == "" {
				return diag.Errorf("values for command may not be empty")
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
	if v, ok := d.GetOk("working_dir"); ok {
		config.WorkingDir = v.(string)
	}
	extraHosts := []string{}
	if v, ok := d.GetOk("host"); ok {
		extraHosts = extraHostsSetToContainerExtraHosts(v.(*schema.Set))
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
			return diag.Errorf("Unable to parse volumes: %s", err)
		}
	}
	if len(volumes) != 0 {
		config.Volumes = volumes
	}

	if v, ok := d.GetOk("labels"); ok {
		config.Labels = labelSetToMap(v.(*schema.Set))
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

	mounts := []mount.Mount{}

	if value, ok := d.GetOk("mounts"); ok {
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

			if mountType == mount.TypeBind {
				if value, ok := rawMount["bind_options"]; ok {
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
			} else if mountType == mount.TypeVolume {
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
			} else if mountType == mount.TypeTmpfs {
				if value, ok := rawMount["tmpfs_options"]; ok {
					if len(value.([]interface{})) > 0 {
						mountInstance.TmpfsOptions = &mount.TmpfsOptions{}
						for _, rawTmpfsOptions := range value.([]interface{}) {
							rawTmpfsOptions := rawTmpfsOptions.(map[string]interface{})
							if value, ok := rawTmpfsOptions["size_bytes"]; ok {
								mountInstance.TmpfsOptions.SizeBytes = (int64)(value.(int))
							}
							if value, ok := rawTmpfsOptions["mode"]; ok {
								mountInstance.TmpfsOptions.Mode = os.FileMode(value.(int))
							}
						}
					}
				}
			}

			mounts = append(mounts, mountInstance)
		}
	}

	hostConfig := &container.HostConfig{
		Privileged:      d.Get("privileged").(bool),
		PublishAllPorts: d.Get("publish_all_ports").(bool),
		RestartPolicy: container.RestartPolicy{
			Name:              d.Get("restart").(string),
			MaximumRetryCount: d.Get("max_retry_count").(int),
		},
		Mounts:         mounts,
		AutoRemove:     d.Get("rm").(bool),
		ReadonlyRootfs: d.Get("read_only").(bool),
		LogConfig: container.LogConfig{
			Type: d.Get("log_driver").(string),
		},
	}

	if v, ok := d.GetOk("tmpfs"); ok {
		hostConfig.Tmpfs = mapTypeMapValsToString(v.(map[string]interface{}))
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

	if v, ok := d.GetOk("security_opts"); ok {
		hostConfig.SecurityOpt = stringSetToStringSlice(v.(*schema.Set))
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

	if v, ok := d.GetOk("shm_size"); ok {
		hostConfig.ShmSize = int64(v.(int)) * 1024 * 1024
	}

	if v, ok := d.GetOk("cpu_shares"); ok {
		hostConfig.CPUShares = int64(v.(int))
	}

	if v, ok := d.GetOk("cpu_set"); ok {
		hostConfig.CpusetCpus = v.(string)
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

	if v, ok := d.GetOk("sysctls"); ok {
		hostConfig.Sysctls = mapTypeMapValsToString(v.(map[string]interface{}))
	}
	if v, ok := d.GetOk("ipc_mode"); ok {
		hostConfig.IpcMode = container.IpcMode(v.(string))
	}
	if v, ok := d.GetOk("group_add"); ok {
		hostConfig.GroupAdd = stringSetToStringSlice(v.(*schema.Set))
	}

	init := d.Get("init").(bool)
	hostConfig.Init = &init

	var retContainer container.ContainerCreateCreatedBody

	// TODO mavogel add platform later which comes from API v1.41. Currently we pass nil
	if retContainer, err = client.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, d.Get("name").(string)); err != nil {
		return diag.Errorf("Unable to create container: %s", err)
	}

	d.SetId(retContainer.ID)

	// Still support the deprecated properties
	if v, ok := d.GetOk("networks"); ok {
		if err := client.NetworkDisconnect(ctx, "bridge", retContainer.ID, false); err != nil {
			if !strings.Contains(err.Error(), "is not connected to the network bridge") {
				return diag.Errorf("Unable to disconnect the default network: %s", err)
			}
		}
		endpointConfig := &network.EndpointSettings{}
		if v, ok := d.GetOk("network_alias"); ok {
			endpointConfig.Aliases = stringSetToStringSlice(v.(*schema.Set))
		}

		for _, rawNetwork := range v.(*schema.Set).List() {
			networkID := rawNetwork.(string)
			if err := client.NetworkConnect(ctx, networkID, retContainer.ID, endpointConfig); err != nil {
				return diag.Errorf("Unable to connect to network '%s': %s", networkID, err)
			}
		}
	}

	// But overwrite them with the future ones, if set
	if v, ok := d.GetOk("networks_advanced"); ok {
		if err := client.NetworkDisconnect(ctx, "bridge", retContainer.ID, false); err != nil {
			if !strings.Contains(err.Error(), "is not connected to the network bridge") {
				return diag.Errorf("Unable to disconnect the default network: %s", err)
			}
		}

		for _, rawNetwork := range v.(*schema.Set).List() {
			networkID := rawNetwork.(map[string]interface{})["name"].(string)

			endpointConfig := &network.EndpointSettings{}
			endpointIPAMConfig := &network.EndpointIPAMConfig{}
			if v, ok := rawNetwork.(map[string]interface{})["aliases"]; ok {
				endpointConfig.Aliases = stringSetToStringSlice(v.(*schema.Set))
			}
			if v, ok := rawNetwork.(map[string]interface{})["ipv4_address"]; ok {
				endpointIPAMConfig.IPv4Address = v.(string)
			}
			if v, ok := rawNetwork.(map[string]interface{})["ipv6_address"]; ok {
				endpointIPAMConfig.IPv6Address = v.(string)
			}
			endpointConfig.IPAMConfig = endpointIPAMConfig

			if err := client.NetworkConnect(ctx, networkID, retContainer.ID, endpointConfig); err != nil {
				return diag.Errorf("Unable to connect to network '%s': %s", networkID, err)
			}
		}
	}

	if v, ok := d.GetOk("upload"); ok {

		var mode int64
		for _, upload := range v.(*schema.Set).List() {
			content := upload.(map[string]interface{})["content"].(string)
			contentBase64 := upload.(map[string]interface{})["content_base64"].(string)
			source := upload.(map[string]interface{})["source"].(string)

			testParams := []string{content, contentBase64, source}
			setParams := 0
			for _, v := range testParams {
				if v != "" {
					setParams++
				}
			}

			if setParams == 0 {
				return diag.Errorf("error with upload content: one of 'content', 'content_base64', or 'source' must be set")
			}
			if setParams > 1 {
				return diag.Errorf("error with upload content: only one of 'content', 'content_base64', or 'source' can be set")
			}

			var contentToUpload string
			if content != "" {
				contentToUpload = content
			}
			if contentBase64 != "" {
				decoded, _ := base64.StdEncoding.DecodeString(contentBase64)
				contentToUpload = string(decoded)
			}
			if source != "" {
				sourceContent, err := ioutil.ReadFile(source)
				if err != nil {
					return diag.Errorf("could not read file: %s", err)
				}
				contentToUpload = string(sourceContent)
			}
			file := upload.(map[string]interface{})["file"].(string)
			executable := upload.(map[string]interface{})["executable"].(bool)

			buf := new(bytes.Buffer)
			tw := tar.NewWriter(buf)
			if executable {
				mode = 0o744
			} else {
				mode = 0o644
			}
			hdr := &tar.Header{
				Name: file,
				Mode: mode,
				Size: int64(len(contentToUpload)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return diag.Errorf("Error creating tar archive: %s", err)
			}
			if _, err := tw.Write([]byte(contentToUpload)); err != nil {
				return diag.Errorf("Error creating tar archive: %s", err)
			}
			if err := tw.Close(); err != nil {
				return diag.Errorf("Error creating tar archive: %s", err)
			}

			dstPath := "/"
			uploadContent := bytes.NewReader(buf.Bytes())
			options := types.CopyToContainerOptions{}
			if err := client.CopyToContainer(ctx, retContainer.ID, dstPath, uploadContent, options); err != nil {
				return diag.Errorf("Unable to upload volume content: %s", err)
			}
		}
	}

	if d.Get("start").(bool) {
		creationTime = time.Now()
		options := types.ContainerStartOptions{}
		if err := client.ContainerStart(ctx, retContainer.ID, options); err != nil {
			return diag.Errorf("Unable to start container: %s", err)
		}
	}

	if d.Get("attach").(bool) {
		var b bytes.Buffer
		logsRead := make(chan bool)
		if d.Get("logs").(bool) {
			go func() {
				defer func() { logsRead <- true }()
				reader, err := client.ContainerLogs(ctx, retContainer.ID, types.ContainerLogsOptions{
					ShowStdout: true,
					ShowStderr: true,
					Follow:     true,
					Timestamps: false,
				})
				if err != nil {
					log.Panic(err)
				}
				defer reader.Close()

				scanner := bufio.NewScanner(reader)
				for scanner.Scan() {
					line := scanner.Text()
					b.WriteString(line)
					b.WriteString("\n")

					log.Printf("[DEBUG] container logs: %s", line)
				}
				if err := scanner.Err(); err != nil {
					log.Fatal(err)
				}
			}()
		}

		attachCh, errAttachCh := client.ContainerWait(ctx, retContainer.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errAttachCh:
			if err != nil {
				return diag.Errorf("Unable to wait container end of execution: %s", err)
			}
		case <-attachCh:
			if d.Get("logs").(bool) {
				// There is a race condition here.
				// If the goroutine does not finish writing into the buffer by this line, we will have no logs.
				// Thus, waiting for the goroutine to finish
				<-logsRead
				d.Set("container_logs", b.String())
			}
		}
	}

	return resourceDockerContainerRead(ctx, d, meta)
}

func resourceDockerContainerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Waiting for container: '%s' to run: max '%v seconds'", d.Id(), containerReadRefreshTimeout)
	client := meta.(*ProviderConfig).DockerClient

	apiContainer, err := fetchDockerContainer(ctx, d.Id(), client)
	if err != nil {
		return diag.FromErr(err)
	}
	if apiContainer == nil {
		// This container doesn't exist anymore
		d.SetId("")
		return nil
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"running"},
		Refresh:    resourceDockerContainerReadRefreshFunc(ctx, d, meta),
		Timeout:    containerReadRefreshTimeout,
		MinTimeout: containerReadRefreshWaitBeforeRefreshes,
		Delay:      containerReadRefreshDelay,
	}

	containerRaw, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		if errors.Is(err, errContainerFailedToBeCreated) {
			return resourceDockerContainerDelete(ctx, d, meta)
		}
		if errors.Is(err, errContainerExitedImmediately) {
			if err := resourceDockerContainerDelete(ctx, d, meta); err != nil {
				log.Printf("[ERROR] Container %s failed to be deleted: %v", apiContainer.ID, err)
				return diag.FromErr(errContainerFailedToBeDeleted)
			}
		}
		return diag.FromErr(err)
	}

	container := containerRaw.(types.ContainerJSON)
	jsonObj, _ := json.MarshalIndent(container, "", "\t")
	log.Printf("[DEBUG] Docker container inspect from stateFunc: %s", jsonObj)

	if !container.State.Running && d.Get("must_run").(bool) {
		if err := resourceDockerContainerDelete(ctx, d, meta); err != nil {
			log.Printf("[ERROR] Container %s failed to be deleted: %v", container.ID, err)
			return err
		}
		log.Printf("[ERROR] Container %s failed to be in running state", container.ID)
		return diag.FromErr(errContainerFailedToBeInRunningState)
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

	// TODO all the other attributes
	d.SetId(container.ID)
	d.Set("name", strings.TrimLeft(container.Name, "/")) // api prefixes with '/' ...
	d.Set("rm", container.HostConfig.AutoRemove)
	d.Set("read_only", container.HostConfig.ReadonlyRootfs)
	// "start" can't be imported
	// attach
	// logs
	// "must_run" can't be imported
	// container_logs
	d.Set("image", container.Image)
	d.Set("hostname", container.Config.Hostname)
	d.Set("domainname", container.Config.Domainname)
	d.Set("command", container.Config.Cmd)
	d.Set("entrypoint", container.Config.Entrypoint)
	d.Set("user", container.Config.User)
	d.Set("dns", container.HostConfig.DNS)
	d.Set("dns_opts", container.HostConfig.DNSOptions)
	d.Set("security_opts", container.HostConfig.SecurityOpt)
	d.Set("dns_search", container.HostConfig.DNSSearch)
	d.Set("publish_all_ports", container.HostConfig.PublishAllPorts)
	d.Set("restart", container.HostConfig.RestartPolicy.Name)
	d.Set("max_retry_count", container.HostConfig.RestartPolicy.MaximumRetryCount)

	// From what I can tell Init being nullable is only for container creation to allow
	// dockerd to default it to the daemons own default settings. So this != nil
	// check is most likely not ever going to fail. In the event that it does the
	// "init" value will be set to false as there isn't much else we can do about it.
	if container.HostConfig.Init != nil {
		d.Set("init", *container.HostConfig.Init)
	} else {
		d.Set("init", false)
	}
	d.Set("working_dir", container.Config.WorkingDir)
	if len(container.HostConfig.CapAdd) > 0 || len(container.HostConfig.CapDrop) > 0 {
		// TODO implement DiffSuppressFunc
		d.Set("capabilities", []interface{}{
			map[string]interface{}{
				"add":  container.HostConfig.CapAdd,
				"drop": container.HostConfig.CapDrop,
			},
		})
	}
	d.Set("mounts", getDockerContainerMounts(container))
	// volumes
	d.Set("tmpfs", container.HostConfig.Tmpfs)
	if err := d.Set("host", flattenExtraHosts(container.HostConfig.ExtraHosts)); err != nil {
		log.Printf("[WARN] failed to set container hostconfig extrahosts from API: %s", err)
	}
	if err = d.Set("ulimit", flattenUlimits(container.HostConfig.Ulimits)); err != nil {
		log.Printf("[WARN] failed to set container hostconfig  ulimits from API: %s", err)
	}

	// We decided not to set the environment variables and labels
	// because they are taken over from the Docker image and aren't scalar
	// so it's difficult to treat them well.
	// For detail, please see the following URLs.
	// https://github.com/terraform-providers/terraform-provider-docker/issues/242
	// https://github.com/terraform-providers/terraform-provider-docker/pull/269

	d.Set("links", container.HostConfig.Links)
	d.Set("privileged", container.HostConfig.Privileged)
	if err = d.Set("devices", flattenDevices(container.HostConfig.Devices)); err != nil {
		log.Printf("[WARN] failed to set container hostconfig devices from API: %s", err)
	}
	// "destroy_grace_seconds" can't be imported
	d.Set("memory", container.HostConfig.Memory/1024/1024)
	if container.HostConfig.MemorySwap > 0 {
		d.Set("memory_swap", container.HostConfig.MemorySwap/1024/1024)
	} else {
		d.Set("memory_swap", container.HostConfig.MemorySwap)
	}
	d.Set("shm_size", container.HostConfig.ShmSize/1024/1024)
	d.Set("cpu_shares", container.HostConfig.CPUShares)
	d.Set("cpu_set", container.HostConfig.CpusetCpus)
	d.Set("log_driver", container.HostConfig.LogConfig.Type)
	d.Set("log_opts", container.HostConfig.LogConfig.Config)
	// "network_alias" is deprecated
	d.Set("network_mode", container.HostConfig.NetworkMode)
	// networks
	// networks_advanced
	d.Set("pid_mode", container.HostConfig.PidMode)
	d.Set("userns_mode", container.HostConfig.UsernsMode)
	// "upload" can't be imported
	if container.Config.Healthcheck != nil {
		d.Set("healthcheck", []interface{}{
			map[string]interface{}{
				"test":         container.Config.Healthcheck.Test,
				"interval":     container.Config.Healthcheck.Interval.String(),
				"timeout":      container.Config.Healthcheck.Timeout.String(),
				"start_period": container.Config.Healthcheck.StartPeriod.String(),
				"retries":      container.Config.Healthcheck.Retries,
			},
		})
	}
	d.Set("sysctls", container.HostConfig.Sysctls)
	d.Set("ipc_mode", container.HostConfig.IpcMode)
	d.Set("group_add", container.HostConfig.GroupAdd)
	d.Set("tty", container.Config.Tty)
	d.Set("stdin_open", container.Config.OpenStdin)

	return nil
}

func resourceDockerContainerReadRefreshFunc(ctx context.Context,
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		containerID := d.Id()

		var container types.ContainerJSON
		container, err := client.ContainerInspect(ctx, containerID)
		if err != nil {
			return container, "pending", err
		}

		jsonObj, _ := json.MarshalIndent(container, "", "\t")
		log.Printf("[DEBUG] Docker container inspect: %s", jsonObj)

		if container.State.Running ||
			!container.State.Running && !d.Get("must_run").(bool) {
			log.Printf("[DEBUG] Container %s is running: %v", containerID, container.State.Running)
			return container, "running", nil
		}

		if creationTime.IsZero() { // We didn't just create it, so don't wait around
			log.Printf("[DEBUG] Container %s was not created", containerID)
			return container, "pending", errContainerFailedToBeCreated
		}

		finishTime, err := time.Parse(time.RFC3339, container.State.FinishedAt)
		if err != nil {
			log.Printf("[ERROR] Container %s finish time could not be parsed: %s", containerID, container.State.FinishedAt)
			return container, "pending", err
		}
		if finishTime.After(creationTime) {
			log.Printf("[INFO] Container %s exited immediately: started: %v - finished: %v", containerID, creationTime, finishTime)
			return container, "pending", errContainerExitedImmediately
		}

		// TODO mavogel wait until all properties are exposed from the API
		// dns               = []
		// dns_opts          = []
		// dns_search        = []
		// group_add         = []
		// id                = "9e6d9e987923e2c3a99f17e8781c7ce3515558df0e45f8ab06f6adb2dda0de50"
		// links             = []
		// log_opts          = {}
		// name              = "nginx"
		// sysctls           = {}
		// tmpfs             = {}

		return container, "running", nil
	}
}

func resourceDockerContainerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	attrs := []string{
		"restart", "max_retry_count", "cpu_shares", "memory", "cpu_set", "memory_swap",
	}
	for _, attr := range attrs {
		if d.HasChange(attr) {

			// TODO update ulimits
			// Updating ulimits seems not to work well.
			// It succeeds to run `DockerClient.ContainerUpdate` with `ulimit` but actually `ulimit` aren't changed.
			// https://github.com/terraform-providers/terraform-provider-docker/pull/236#discussion_r373819536
			// ulimits := []*units.Ulimit{}
			// if v, ok := d.GetOk("ulimit"); ok {
			// 	ulimits = ulimitsToDockerUlimits(v.(*schema.Set))
			// }

			updateConfig := container.UpdateConfig{
				RestartPolicy: container.RestartPolicy{
					Name:              d.Get("restart").(string),
					MaximumRetryCount: d.Get("max_retry_count").(int),
				},
				Resources: container.Resources{
					CPUShares:  int64(d.Get("cpu_shares").(int)),
					Memory:     int64(d.Get("memory").(int)) * 1024 * 1024,
					CpusetCpus: d.Get("cpu_set").(string),
					// Ulimits:    ulimits,
				},
			}

			if ms, ok := d.GetOk("memory_swap"); ok {
				a := int64(ms.(int))
				if a > 0 {
					a = a * 1024 * 1024
				}
				updateConfig.Resources.MemorySwap = a
			}
			client := meta.(*ProviderConfig).DockerClient
			_, err := client.ContainerUpdate(ctx, d.Id(), updateConfig)
			if err != nil {
				return diag.Errorf("Unable to update a container: %v", err)
			}
			break
		}
	}
	return nil
}

func resourceDockerContainerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient

	if d.Get("rm").(bool) {
		d.SetId("")
		return nil
	}

	if !d.Get("attach").(bool) {
		// Stop the container before removing if destroy_grace_seconds is defined
		if d.Get("destroy_grace_seconds").(int) > 0 {
			timeout := time.Duration(int32(d.Get("destroy_grace_seconds").(int))) * time.Second

			if err := client.ContainerStop(ctx, d.Id(), &timeout); err != nil {
				return diag.Errorf("Error stopping container %s: %s", d.Id(), err)
			}
		}
	}

	removeOpts := types.ContainerRemoveOptions{
		RemoveVolumes: d.Get("remove_volumes").(bool),
		Force:         true,
	}

	if err := client.ContainerRemove(ctx, d.Id(), removeOpts); err != nil {
		return diag.Errorf("Error deleting container %s: %s", d.Id(), err)
	}

	waitOkC, errorC := client.ContainerWait(ctx, d.Id(), container.WaitConditionRemoved)
	select {
	case waitOk := <-waitOkC:
		log.Printf("[INFO] Container exited with code [%v]: '%s'", waitOk.StatusCode, d.Id())
	case err := <-errorC:
		if !(strings.Contains(err.Error(), "No such container") || strings.Contains(err.Error(), "is already in progress")) {
			return diag.Errorf("Error waiting for container removal '%s': %s", d.Id(), err)
		}
	}

	d.SetId("")
	return nil
}

func fetchDockerContainer(ctx context.Context, ID string, client *client.Client) (*types.Container, error) {
	apiContainers, err := client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("error fetching container information from Docker: %s\n", err)
	}

	for _, apiContainer := range apiContainers {
		if apiContainer.ID == ID {
			return &apiContainer, nil
		}
	}

	return nil, nil
}
