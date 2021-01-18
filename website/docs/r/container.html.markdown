---
layout: "docker"
page_title: "Docker: docker_container"
sidebar_current: "docs-docker-resource-container"
description: |-
  Manages the lifecycle of a Docker container.
---

# docker\_container

Manages the lifecycle of a Docker container.

## Example Usage

```hcl
# Start a container
resource "docker_container" "ubuntu" {
  name  = "foo"
  image = "${docker_image.ubuntu.latest}"
}

# Find the latest Ubuntu precise image.
resource "docker_image" "ubuntu" {
  name = "ubuntu:precise"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the Docker container.
* `image` - (Required, string) The ID of the image to back this container.
  The easiest way to get this value is to use the `docker_image` resource
  as is shown in the example above.

* `command` - (Optional, list of strings) The command to use to start the
    container. For example, to run `/usr/bin/myprogram -f baz.conf` set the
    command to be `["/usr/bin/myprogram", "-f", "baz.conf"]`.
* `entrypoint` - (Optional, list of strings) The command to use as the
    Entrypoint for the container. The Entrypoint allows you to configure a
    container to run as an executable. For example, to run `/usr/bin/myprogram`
    when starting a container, set the entrypoint to be
    `["/usr/bin/myprogram"]`.
* `user` - (Optional, string) User used for run the first process. Format is
    `user` or `user:group` which user and group can be passed literraly or
    by name.
* `dns` - (Optional, set of strings) Set of DNS servers.
* `dns_opts` - (Optional, set of strings) Set of DNS options used by the DNS provider(s), see `resolv.conf` documentation for valid list of options.
* `dns_search` - (Optional, set of strings) Set of DNS search domains that are used when bare unqualified hostnames are used inside of the container.
* `env` - (Optional, set of strings) Environment variables to set.
* `labels` - (Optional, block) See [Labels](#labels-1) below for details.
* `links` - (Optional, set of strings) Set of links for link based
  connectivity between containers that are running on the same host.

~> **Warning** The --link flag is a legacy feature of Docker. It may eventually
be removed. It exposes _all_ environment variables originating from Docker to
any linked containers. This could have serious security implications if sensitive
data is stored in them. See [the docker documentation](https://docs.docker.com/network/links/) for more details.

* `hostname` - (Optional, string) Hostname of the container.
* `domainname` - (Optional, string) Domain name of the container.
* `restart` - (Optional, string) The restart policy for the container. Must be
  one of "no", "on-failure", "always", "unless-stopped".
* `max_retry_count` - (Optional, int) The maximum amount of times to an attempt
  a restart when `restart` is set to "on-failure"
* `working_dir`- (Optional, string) The working directory for commands to run in
* `rm` - (Optional, boolean) If true, then the container will be automatically removed after his execution. Terraform
   won't check this container after creation.
* `read_only` - (Optional, boolean) If true, the container will be started as readonly.
* `start` - (Optional, boolean) If true, then the Docker container will be
  started after creation. If false, then the container is only created.
* `attach` - (Optional, boolean) If true attach to the container after its creation and waits the end of his execution.
* `logs` - (Optional, boolean) Save the container logs (`attach` must be enabled).
* `must_run` - (Optional, boolean) If true, then the Docker container will be
  kept running. If false, then as long as the container exists, Terraform
  assumes it is successful.
* `tty` - (Optional, boolean) if true, allocate a pseudo-tty (docker run -t)
* `stdin_open` - (Optional, boolean) if true, keep STDIN open even if not attached (docker run -i)
* `capabilities` - (Optional, block) See [Capabilities](#capabilities-1) below for details.
* `security_opts` - (Optional, set of strings) Set of string values to customize labels for MLS systems, such as SELinux. See https://docs.docker.com/engine/reference/run/#security-configuration.
* `mounts` - (Optional, set of blocks) See [Mounts](#mounts-1) below for details.
* `tmpfs` - (Optional, map) A map of container directories which should be replaced by `tmpfs mounts`, and their corresponding mount options.
* `ports` - (Optional, block) See [Ports](#ports-1) below for details.
* `host` - (Optional, block) See [Extra Hosts](#extra_hosts-1) below for
  details.
* `privileged` - (Optional, boolean) Run container in privileged mode.
* `devices` - (Optional, block) See [Devices](#devices-1) below for details.
* `publish_all_ports` - (Optional, boolean) Publish all ports of the container.
* `volumes` - (Optional, block) See [Volumes](#volumes-1) below for details.
* `memory` - (Optional, int) The memory limit for the container in MBs.
* `memory_swap` - (Optional, int) The total memory limit (memory + swap) for the
  container in MBs. This setting may compute to `-1` after `terraform apply` if the target host doesn't support memory swap, when that is the case docker will use a soft limitation.
* `shm_size` - (Optional, int) Size of `/dev/shm` in MBs.
* `cpu_shares` - (Optional, int) CPU shares (relative weight) for the container.
* `cpu_set` - (Optional, string) A comma-separated list or hyphen-separated range of CPUs a container can use, e.g. `0-1`.
* `log_driver` - (Optional, string) The logging driver to use for the container.
  Defaults to "json-file".
* `log_opts` - (Optional, map of strings) Key/value pairs to use as options for
  the logging driver.
* `network_alias` - (Optional, set of strings) Network aliases of the container for user-defined networks only. *Deprecated:* use `networks_advanced` instead.
* `network_mode` - (Optional, string) Network mode of the container.
* `networks` - (Optional, set of strings) Id of the networks in which the
  container is. *Deprecated:* use `networks_advanced` instead.
* `networks_advanced` - (Optional, block) See [Networks Advanced](#networks_advanced-1) below for details. If this block has priority to the deprecated `network_alias` and `network` properties.
* `destroy_grace_seconds` - (Optional, int) If defined will attempt to stop the container before destroying. Container will be destroyed after `n` seconds or on successful stop.
* `upload` - (Optional, block) See [File Upload](#upload-1) below for details.
* `ulimit` - (Optional, block) See [Ulimits](#ulimits-1) below for
  details.
* `pid_mode` - (Optional, string) The PID (Process) Namespace mode for the container. Either `container:<name|id>` or `host`.
* `userns_mode` - (Optional, string) Sets the usernamespace mode for the container when usernamespace remapping option is enabled.
* `healthcheck` - (Optional, block) See [Healthcheck](#healthcheck-1) below for details.
* `sysctls` - (Optional, map) A map of kernel parameters (sysctls) to set in the container.
* `ipc_mode` - (Optional, string) IPC sharing mode for the container. Possible values are: `none`, `private`, `shareable`, `container:<name|id>` or `host`.
* `group_add` - (Optional, set of strings) Add additional groups to run as.
* `init` - (Optional, bool) Configured whether an init process should be injected for this container. If unset this will default to the `dockerd` defaults.

<a id="labels-1"></a>
#### Labels

`labels` is a block within the configuration that can be repeated to specify
additional label name and value data to the container. Each `labels` block supports
the following:

* `label` - (Required, string) Name of the label
* `value` (Required, string) Value of the label

See [214](https://github.com/terraform-providers/terraform-provider-docker/issues/214#issuecomment-550128950) for Details.

<a id="capabilities-1"></a>
### Capabilities

`capabilities` is a block within the configuration that allows you to add or drop linux capabilities. For more information about what capabilities you can add and drop please visit the docker run documentation.

* `add` - (Optional, set of strings) list of linux capabilities to add.
* `drop` - (Optional, set of strings) list of linux capabilities to drop.

Example:

```hcl
resource "docker_container" "ubuntu" {
  name  = "foo"
  image = "${docker_image.ubuntu.latest}"

  capabilities {
    add  = ["ALL"]
    drop = ["SYS_ADMIN"]
  }
}
```

<a id="mounts-1"></a>
### Mounts

`mounts` is a block within the configuration that can be repeated to specify
the extra mount mappings for the container. Each `mounts` block is the Specification for mounts to be added to container and
supports the following:

* `target` - (Required, string) The container path.
* `source` - (Optional, string) The mount source (e.g., a volume name, a host path)
* `type` - (Required, string) The mount type: valid values are `bind|volume|tmpfs`.
* `read_only` - (Optional, string) Whether the mount should be read-only
* `bind_options` - (Optional, map) Optional configuration for the `bind` type.
  * `propagation` - (Optional, string) A propagation mode with the value.
* `volume_options` - (Optional, map) Optional configuration for the `volume` type.
  * `no_copy` - (Optional, string) Whether to populate volume with data from the target.
  * `labels` - (Optional, map of key/value pairs) Adding labels.
  * `driver_options` - (Optional, map of key/value pairs) Options for the driver.
* `tmpfs_options` - (Optional, map) Optional configuration for the `tmpf` type.
  * `size_bytes` - (Optional, int) The size for the tmpfs mount in bytes.
  * `mode` - (Optional, int) The permission mode for the tmpfs mount in an integer.

<a id="ports-1"></a>
### Ports

`ports` is a block within the configuration that can be repeated to specify
the port mappings of the container. Each `ports` block supports
the following:

* `internal` - (Required, int) Port within the container.
* `external` - (Optional, int) Port exposed out of the container. If not given a free random port `>= 32768` will be used.
* `ip` - (Optional, string) IP address/mask that can access this port, default to `0.0.0.0`
* `protocol` - (Optional, string) Protocol that can be used over this port,
  defaults to `tcp`.

<a id="extra_hosts"></a>
### Extra Hosts

`host` is a block within the configuration that can be repeated to specify
the extra host mappings for the container. Each `host` block supports
the following:

* `host` - (Required, string) Hostname to add.
* `ip` - (Required, string) IP address this hostname should resolve to.

This is equivalent to using the `--add-host` option when using the `run`
command of the Docker CLI.

<a id="volumes-1"></a>
### Volumes

`volumes` is a block within the configuration that can be repeated to specify
the volumes attached to a container. Each `volumes` block supports
the following:

* `from_container` - (Optional, string) The container where the volume is
  coming from.
* `host_path` - (Optional, string) The path on the host where the volume
  is coming from.
* `volume_name` - (Optional, string) The name of the docker volume which
  should be mounted.
* `container_path` - (Optional, string) The path in the container where the
  volume will be mounted.
* `read_only` - (Optional, bool) If true, this volume will be readonly.
  Defaults to false.

One of `from_container`, `host_path` or `volume_name` must be set.

<a id="upload-1"></a>
### File Upload

`upload` is a block within the configuration that can be repeated to specify
files to upload to the container before starting it. Only one of `content` or `content_base64` can be set and at least
one of them has to be set.
Each `upload` supports the following

* `content` - (Optional, string, conflicts with `content_base64` & `source`) Literal string value to use as the object content, which will be uploaded as UTF-8-encoded text.
* `content_base64` - (Optional, string, conflicts with `content` & `source`) Base64-encoded data that will be decoded and uploaded as raw bytes for the object content. This allows safely uploading non-UTF8 binary data, but is recommended only for larger binary content such as the result of the `base64encode` interpolation function. See [here](https://github.com/terraform-providers/terraform-provider-docker/issues/48#issuecomment-374174588) for the reason.
* `source` - (Optional, string, conflicts with `content` & `content_base64`) A filename that references a file which will be uploaded as the object content. This allows for large file uploads that do not get stored in state.
* `source_hash` - (Optional, string) If using `source`, this will force an update if the file content has updated but the filename has not. 
* `file` - (Required, string) path to a file in the container.
* `executable` - (Optional, boolean) If true, the file will be uploaded with user
  executable permission.
  Defaults to false.

<a id="networks_advanced-1"></a>
### Networks advanced

`networks_advanced` is a block within the configuration that can be repeated to specify
advanced options for the container in a specific network.
Each `networks_advanced` supports the following:

* `name` - (Required, string) The name of the network.
* `aliases` - (Optional, set of strings) The network aliases of the container in the specific network.
* `ipv4_address` - (Optional, string) The IPV4 address of the container in the specific network.
* `ipv6_address` - (Optional, string) The IPV6 address of the container in the specific network.

<a id="devices-1"></a>
### Devices

`devices` is a block within the configuration that can be repeated to specify
the devices exposed to a container. Each `devices` block supports
the following:

* `host_path` - (Required, string) The path on the host where the device
  is located.
* `container_path` - (Optional, string) The path in the container where the
  device will be binded.
* `permissions` - (Optional, string) The cgroup permissions given to the
  container to access the device.
  Defaults to `rwm`.

<a id="ulimits-1"></a>
### Ulimits

`ulimit` is a block within the configuration that can be repeated to specify
the extra ulimits for the container. Each `ulimit` block supports
the following:

* `name` - (Required, string)
* `soft` - (Required, int)
* `hard` - (Required, int)

<a id="healthcheck-1"></a>
### Healthcheck

`healthcheck` is a block within the configuration that can be repeated only **once** to specify the extra healthcheck configuration for the container. The `healthcheck` block is a test to perform to check that the container is healthy and supports the following:

* `test` - (Required, list of strings) Command to run to check health. For example, to run `curl -f http://localhost/health` set the
    command to be `["CMD", "curl", "-f", "http://localhost/health"]`.
* `interval` - (Optional, string) Time between running the check `(ms|s|m|h)`. Default: `0s`.
* `timeout` - (Optional, string) Maximum time to allow one check to run `(ms|s|m|h)`. Default: `0s`.
* `start_period` - (Optional, string) Start period for the container to initialize before counting retries towards unstable `(ms|s|m|h)`. Default: `0s`.
* `retries` - (Optional, int) Consecutive failures needed to report unhealthy. Default: `0`.

## Attributes Reference

The following attributes are exported:

 * `exit_code` - The exit code of the container if its execution is done (`must_run` must be disabled).
 * `container_logs` - The logs of the container if its execution is done (`attach` must be disabled).
 * `network_data` - (Map of a block) The IP addresses of the container on each
   network. Key are the network names, values are the IP addresses.
   * `ip_address` - The IP address of the container.
   * `ip_prefix_length` - The IP prefix length of the container.
   * `gateway` - The network gateway of the container.
 * `bridge` - The network bridge of the container as read from its NetworkSettings.
 * `ip_address` - *Deprecated:* Use `network_data` instead. The IP address of the container's first network it.
 * `ip_prefix_length` - *Deprecated:* Use `network_data` instead. The IP prefix length of the container as read from its
   NetworkSettings.
 * `gateway` - *Deprecated:* Use `network_data` instead. The network gateway of the container as read from its
   NetworkSettings.

## Import

Docker containers can be imported using the long id, e.g. for a container named `foo`:

```sh
$ terraform import docker_container.foo $(docker inspect -f {{.ID}} foo)
```
