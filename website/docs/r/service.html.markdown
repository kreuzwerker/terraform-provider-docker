---
layout: "docker"
page_title: "Docker: docker_service"
sidebar_current: "docs-docker-resource-service"
description: |-
  Manages the lifecycle of a Docker service.
---

# docker\_service

This resource manages the lifecycle of a Docker service. By default, the creation, update and delete of services are detached.

With the [Converge Config](#convergeconfig) the behavior of the `docker cli` is imitated to guarantee that
for example, all tasks of a service are running or successfully updated or to inform `terraform` that a service could not
be updated and was successfully rolled back.

## Example Usage
The following examples show the basic and advanced usage of the
Docker Service resource assuming the host machine is already part of a Swarm.

### Basic
The following configuration starts a Docker Service with 
- the given image, 
- 1 replica
- exposes the port `8080` in `vip` mode to the host machine
- moreover, uses the `container` runtime

```hcl
resource "docker_service" "foo" {
  name = "foo-service"

  task_spec {
    container_spec {
      image = "repo.mycompany.com:8080/foo-service:v1"
    }
  }

  endpoint_spec {
    ports {
      target_port = "8080"
    }
  }
}
```

The following command is the equivalent:

```bash
$ docker service create -d -p 8080 --name foo-service repo.mycompany.com:8080/foo-service:v1
```

### Advanced
The following configuration shows the full capabilities of a Docker Service. Currently, the [Docker API 1.32](https://docs.docker.com/engine/api/v1.32) is implemented.

```hcl
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
      image = "repo.mycompany.com:8080/foo-service:v1"

      labels {
        label = "foo.bar"
        value = "baz"
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

      mounts {
          target    = "/mount/test"
          source    = "${docker_volume.test_volume.name}"
          type      = "volume"
          read_only = true

          bind_options {
            propagation = "private"
          }
        }
      

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
          file_name   = "/secrets.json"
          file_uid    = "0"
					file_gid    = "0"
					file_mode   = 0777
        },
      ]

      configs = [
        {
          config_id   = "${docker_config.service_config.id}"
          config_name = "${docker_config.service_config.name}"
          file_name   = "/configs.json"
        },
      ]
    }

    resources {
      limits {
        nano_cpus    = 1000000
        memory_bytes = 536870912

        generic_resources {
          named_resources_spec = [
            "GPU=UUID1",
          ]

          discrete_resources_spec = [
            "SSD=3",
          ]
        }
      }

      reservation {
        nano_cpus    = 1000000
        memory_bytes = 536870912

        generic_resources {
          named_resources_spec = [
            "GPU=UUID1",
          ]

          discrete_resources_spec = [
            "SSD=3",
          ]
        }
      }
    }

    restart_policy = {
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
```

See also the `TestAccDockerService_full` test or all the other tests for a complete overview.

## Argument Reference

The following arguments are supported:

* `auth` - (Optional, block) See [Auth](#auth-1) below for details.
* `name` - (Required, string) The name of the Docker service.
* `task_spec` - (Required, block) See [TaskSpec](#task-spec-1) below for details.
* `mode` - (Optional, block) See [Mode](#mode-1) below for details.
* `update_config` - (Optional, block) See [UpdateConfig](#update-rollback-config-1) below for details.
* `rollback_config` - (Optional, block) See [RollbackConfig](#update-rollback-config-1) below for details.
* `endpoint_spec` - (Optional, block) See [EndpointSpec](#endpoint-spec-1) below for details.
* `converge_config` - (Optional, block) See [Converge Config](#converge-config-1) below for details.

<a id="auth-1"></a>
### Auth

`auth` can be used additionally to the `registry_auth`. If both properties are given the `auth` wins and overwrites the auth of the provider.

* `server_address` - (Required, string) The address of the registry server
* `username` - (Optional, string) The username to use for authenticating to the registry. If this is blank, the `DOCKER_REGISTRY_USER` is also be checked. 
* `password` - (Optional, string) The password to use for authenticating to the registry. If this is blank, the `DOCKER_REGISTRY_PASS` is also be checked.

<!-- start task-spec -->
<a id="task-spec-1"></a>
### TaskSpec

`task_spec` is a block within the configuration that can be repeated only **once** to specify the mode configuration for the service. The `task_spec` block is the user modifiable task configuration and supports the following:

* `container_spec` (Required, block) See [ContainerSpec](#container-spec-1) below for details.
* `resources` (Optional, block) See [Resources](#resources-1) below for details.
* `restart_policy` (Optional, block) See [Restart Policy](#restart-policy-1) below for details.
* `placement` (Optional, block) See [Placement](#placement-1) below for details.
* `force_update` (Optional, int) A counter that triggers an update even if no relevant parameters have been changed. See [Docker Spec](https://github.com/docker/swarmkit/blob/master/api/specs.proto#L126).
* `runtime` (Optional, string) Runtime is the type of runtime specified for the task executor. See [Docker Runtime](https://github.com/moby/moby/blob/master/api/types/swarm/runtime.go).
* `networks` - (Optional, set of strings) Ids of the networks in which the container will be put in.
* `log_driver` - (Optional, block) See [Log Driver](#log-driver-1) below for details.


<!-- start task-container-spec -->
<a id="container-spec-1"></a>
#### ContainerSpec

`container_spec` is a block within the configuration that can be repeated only **once** to specify the mode configuration for the service. The `container_spec` block is the spec for each container and supports the following:

* `image` - (Required, string) The image used to create the Docker service.
* `labels` - (Optional, block) See [Labels](#labels-1) below for details.
* `command` - (Optional, list of strings) The command to be run in the image.
* `args` - (Optional, list of strings) Arguments to the command.
* `hostname` - (Optional, string) The hostname to use for the container, as a valid RFC 1123 hostname.
* `env` - (Optional, map of string/string) A list of environment variables in the form VAR=value.
* `dir` - (Optional, string) The working directory for commands to run in.
* `user` - (Optional, string) The user inside the container.
* `groups` - (Optional, list of strings) A list of additional groups that the container process will run as.
* `privileges` (Optional, block) See [Privileges](#privileges-1) below for details.
* `read_only` - (Optional, bool) Mount the container's root filesystem as read only.
* `mounts` - (Optional, set of blocks) See [Mounts](#mounts-1) below for details.
* `stop_signal` - (Optional, string) Signal to stop the container.
* `stop_grace_period` - (Optional, string) Amount of time to wait for the container to terminate before forcefully removing it `(ms|s|m|h)`.
* `healthcheck` - (Optional, block) See [Healthcheck](#healthcheck-1) below for details.
* `host` - (Optional, map of string/string) A list of hostname/IP mappings to add to the container's hosts file.
  * `ip` - (Required string) The ip
  * `host` - (Required string) The hostname
* `dns_config` - (Optional, block) See [DNS Config](#dnsconfig-1) below for details.
* `secrets` - (Optional, set of blocks) See [Secrets](#secrets-1) below for details.
* `configs` - (Optional, set of blocks) See [Configs](#configs-1) below for details.
* `isolation` - (Optional, string) Isolation technology of the containers running the service. (Windows only). Valid values are: `default|process|hyperv`


<a id="labels-1"></a>
#### Labels

`labels` is a block within the configuration that can be repeated to specify
additional label name and value data to the container. Each `labels` block supports
the following:

* `label` - (Required, string) Name of the label
* `value` (Required, string) Value of the label

See [214](https://github.com/terraform-providers/terraform-provider-docker/issues/214#issuecomment-550128950) for Details.

<a id="privileges-1"></a>
#### Privileges

`privileges` is a block within the configuration that can be repeated only **once** to specify the mode configuration for the service. The `privileges` block holds the security options for the container and supports the following:

* `credential_spec` - (Optional, block) For managed service account (Windows only)
  * `file` - (Optional, string) Load credential spec from this file.
  * `registry` - (Optional, string) Load credential spec from this value in the Windows registry.
* `se_linux_context` - (Optional, block) SELinux labels of the container
  * `disable` - (Optional, bool) Disable SELinux
  * `user` - (Optional, string) SELinux user label
  * `role` - (Optional, string) SELinux role label
  * `type` - (Optional, string) SELinux type label
  * `level` - (Optional, string) SELinux level label

<a id="mounts-1"></a>
#### Mounts

`mounts` is a block within the configuration that can be repeated to specify
the extra mount mappings for the container. Each `mounts` block is the Specification for mounts to be added to containers created as part of the service and supports
the following:

* `target` - (Required, string) The container path.
* `source` - (Optional, string) The mount source (e.g., a volume name, a host path)
* `type` - (Required, string) The mount type: valid values are `bind|volume|tmpfs`.
* `read_only` - (Optional, string) Whether the mount should be read-only
* `bind_options` - (Optional, map) Optional configuration for the `bind` type.
  * `propagation` - (Optional, string) A propagation mode with the value.
* `volume_options` - (Optional, map) Optional configuration for the `volume` type.
  * `no_copy` - (Optional, string) Whether to populate volume with data from the target.
  * `labels` - (Optional, block) See [Labels](#labels-1) above for details.
  * `driver_config` - (Optional, map) The name of the driver to create the volume.
    * `name` - (Optional, string) The name of the driver to create the volume.
    * `options` - (Optional, map of key/value pairs) Options for the driver.
* `tmpfs_options` - (Optional, map) Optional configuration for the `tmpf` type.
  * `size_bytes` - (Optional, int) The size for the tmpfs mount in bytes. 
  * `mode` - (Optional, int) The permission mode for the tmpfs mount in an integer.

<a id="healthcheck-1"></a>
#### Healthcheck

`healthcheck` is a block within the configuration that can be repeated only **once** to specify the extra healthcheck configuration for the containers of the service. The `healthcheck` block is a test to perform to check that the container is healthy and supports the following:

* `test` - (Required, list of strings) Command to run to check health. For example, to run `curl -f http://localhost/health` set the
    command to be `["CMD", "curl", "-f", "http://localhost/health"]`.
* `interval` - (Optional, string) Time between running the check `(ms|s|m|h)`. Default: `0s`.
* `timeout` - (Optional, string) Maximum time to allow one check to run `(ms|s|m|h)`. Default: `0s`.
* `start_period` - (Optional, string) Start period for the container to initialize before counting retries towards unstable `(ms|s|m|h)`. Default: `0s`.
* `retries` - (Optional, int) Consecutive failures needed to report unhealthy. Default: `0`.

<a id="dnsconfig-1"></a>
### DNS Config

`dns_config` is a block within the configuration that can be repeated only **once** to specify the extra DNS configuration for the containers of the service. The `dns_config` block supports the following:

* `nameservers` - (Required, list of strings) The IP addresses of the name servers, for example, `8.8.8.8`
* `search` - (Optional, list of strings)A search list for host-name lookup.
* `options` - (Optional, list of strings) A list of internal resolver variables to be modified, for example, `debug`, `ndots:3`

<a id="secrets-1"></a>
### Secrets

`secrets` is a block within the configuration that can be repeated to specify
the extra mount mappings for the container. Each `secrets` block is a reference to a secret that will be exposed to the service and supports the following:

* `secret_id` - (Required, string) ConfigID represents the ID of the specific secret.
* `secret_name` - (Optional, string) The name of the secret that this references, but internally it is just provided for lookup/display purposes
* `file_name` - (Required, string) Represents the final filename in the filesystem. The specific target file that the secret data is written within the docker container, e.g. `/root/secret/secret.json`
* `file_uid` - (Optional, string) Represents the file UID. Defaults: `0`
* `file_gid` - (Optional, string) Represents the file GID. Defaults: `0`
* `file_mode` - (Optional, int) Represents the FileMode of the file. Defaults: `0444`

<a id="configs-1"></a>
### Configs

`configs` is a block within the configuration that can be repeated to specify
the extra mount mappings for the container. Each `configs` is a reference to a secret that is exposed to the service and supports the following:

* `config_id` - (Required, string) ConfigID represents the ID of the specific config.
* `config_name` - (Optional, string) The name of the config that this references, but internally it is just provided for lookup/display purposes
* `file_name` - (Required, string) Represents the final filename in the filesystem. The specific target file that the config data is written within the docker container, e.g. `/root/config/config.json`
* `file_uid` - (Optional, string) Represents the file UID. Defaults: `0`
* `file_gid` - (Optional, string) Represents the file GID. Defaults: `0`
* `file_mode` - (Optional, int) Represents the FileMode of the file. Defaults: `0444`

<!-- end task-container-spec -->

<!-- start task-resources-spec -->
<a id="resources-1"></a>
#### Resources

`resources` is a block within the configuration that can be repeated only **once** to specify the mode configuration for the service. The `resources` block represents the requirements which apply to each container created as part of the service and supports the following:

* `limits` - (Optional, list of strings) Describes the resources which can be advertised by a node and requested by a task.
  * `nano_cpus` (Optional, int) CPU shares in units of 1/1e9 (or 10^-9) of the CPU. Should be at least 1000000
  * `memory_bytes` (Optional, int) The amount of memory in bytes the container allocates
  * `generic_resources` (Optional, map) User-defined resources can be either Integer resources (e.g, SSD=3) or String resources (e.g, GPU=UUID1)
    * `named_resources_spec` (Optional, set of string) The String resources, delimited by `=`
    * `discrete_resources_spec` (Optional, set of string) The Integer resources, delimited by `=`
* `reservation` - (Optional, list of strings) An object describing the resources which can be advertised by a node and requested by a task.
  * `nano_cpus` (Optional, int) CPU shares in units of 1/1e9 (or 10^-9) of the CPU. Should be at least 1000000
  * `memory_bytes` (Optional, int) The amount of memory in bytes the container allocates
  * `generic_resources` (Optional, map) User-defined resources can be either Integer resources (e.g, SSD=3) or String resources (e.g, GPU=UUID1)
    * `named_resources_spec` (Optional, set of string) The String resources
    * `discrete_resources_spec` (Optional, set of string) The Integer resources

<!-- end task-resources-spec -->
<!-- start task-restart-policy-spec -->
<a id="restart_policy-1"></a>
#### Restart Policy

`restart_policy` is a block within the configuration that can be repeated only **once** to specify the mode configuration for the service. The `restart_policy` block specifies the restart policy which applies to containers created as part of this service and supports the following:

* `condition` (Optional, string) Condition for restart: `(none|on-failure|any)`
* `delay` (Optional, string) Delay between restart attempts `(ms|s|m|h)`
* `max_attempts` (Optional, string) Maximum attempts to restart a given container before giving up (default value is `0`, which is ignored)
* `window` (Optional, string) The time window used to evaluate the restart policy (default value is `0`, which is unbounded) `(ms|s|m|h)`

<!-- end task-restart-policy-spec -->
<!-- start task-placement-spec -->
<a id="placement-1"></a>
#### Placement

`placement` is a block within the configuration that can be repeated only **once** to specify the mode configuration for the service. The `placement` block specifies the placement preferences and supports the following:

* `constraints` (Optional, set of strings) An array of constraints. e.g.: `node.role==manager`
* `prefs` (Optional, set of string) Preferences provide a way to make the scheduler aware of factors such as topology. They are provided in order from highest to lowest precedence, e.g.: `spread=node.role.manager`
* `platforms` (Optional, set of) Platforms stores all the platforms that the service's image can run on
  * `architecture` (Required, string) The architecture, e.g., `amd64`
  * `os` (Required, string) The operation system, e.g., `linux`

<!-- end task-placement-spec -->
<!-- end log-driver-spec -->
<a id="log-driver-1"></a>
### Log Driver

`log_driver` is a block within the configuration that can be repeated only **once** to specify the extra log_driver configuration for the containers of the service. The `log_driver` specifies the log driver to use for tasks created from this spec. If not present, the default one for the swarm will be used, finally falling back to the engine default if not specified. The block supports the following:

* `name` - (Required, string) The logging driver to use. Either `(none|json-file|syslog|journald|gelf|fluentd|awslogs|splunk|etwlogs|gcplogs)`.
* `options` - (Optional, a map of strings and strings) The options for the logging driver, e.g.

```hcl
options {
  awslogs-region = "us-west-2"
  awslogs-group  = "dev/foo-service"
}
```
<!-- end log-driver-spec -->
<!-- end task-spec -->

<a id="mode-1"></a>
### Mode

`mode` is a block within the configuration that can be repeated only **once** to specify the mode configuration for the service. The `mode` block supports the following:

* `global` - (Optional, bool) set it to `true` to run the service in the global mode

```hcl
resource "docker_service" "foo" {
  ...
  mode {
    global = true
  }
  ...
}
```
* `replicated` - (Optional, map), which contains atm only the amount of `replicas`

```hcl
resource "docker_service" "foo" {
  ...
  mode {
    replicated {
      replicas = 2
    }
  }
  ...
}
```

~> **NOTE on `mode`:** if neither `global` nor `replicated` is specified, the service
is started in `replicated` mode with 1 replica. A change of service mode is not possible. The service has to be destroyed an recreated in the new mode.

<a id="update-rollback-config-1"></a>
### UpdateConfig and RollbackConfig

`update_config` or `rollback_config` is a block within the configuration that can be repeated only **once** to specify the extra update configuration for the containers of the service. The `update_config` `rollback_config` block supports the following:

* `parallelism` - (Optional, int) The maximum number of tasks to be updated in one iteration simultaneously (0 to update all at once).
* `delay` - (Optional, int) Delay between updates `(ns|us|ms|s|m|h)`, e.g. `5s`.
* `failure_action` - (Optional, int) Action on update failure: `pause|continue|rollback`.
* `monitor` - (Optional, int) Duration after each task update to monitor for failure `(ns|us|ms|s|m|h)`
* `max_failure_ratio` - (Optional, string) The failure rate to tolerate during an update as `float`. **Important:** the `float`need to be wrapped in a `string` to avoid internal
casting and precision errors.
* `order` - (Optional, int) Update order either 'stop-first' or 'start-first'.

<a id="endpoint-spec-1"></a>
### EndpointSpec

`endpoint_spec` is a block within the configuration that can be repeated only **once** to specify properties that can be configured to access and load balance a service. The block supports the following:

* `mode` - (Optional, string) The mode of resolution to use for internal load balancing between tasks. `(vip|dnsrr)`. Default: `vip`.
* `ports` - (Optional, block) See [Ports](#ports-1) below for details.

<a id="ports-1"></a>
#### Ports

`ports` is a block within the configuration that can be repeated to specify
the port mappings of the container. Each `ports` block supports
the following:

* `name` - (Optional, string) A random name for the port.
* `protocol` - (Optional, string) Protocol that can be used over this port: `tcp|udp|sctp`. Default: `tcp`.
* `target_port` - (Required, int) Port inside the container.
* `published_port` - (Required, int) The port on the swarm hosts. If not set the value of `target_port` will be used.
* `publish_mode` - (Optional, string) Represents the mode in which the port is to be published: `ingress|host`

<a id="converge-config-1"></a>
### Converge Config

`converge_config` is a block within the configuration that can be repeated only **once** to specify the extra Converging configuration for the containers of the service. This is the same behavior as the `docker cli`. By adding this configuration, it is monitored with the
given interval that, e.g., all tasks/replicas of a service are up and healthy

The `converge_config` block supports the following:

* `delay` - (Optional, string) Time between each the check to check docker endpoint `(ms|s|m|h)`. For example, to check if
all tasks are up when a service is created, or to check if all tasks are successfully updated on an update. Default: `7s`.
* `timeout` - (Optional, string) The timeout of the service to reach the desired state `(s|m)`. Default: `3m`.

## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `id` (string)

## Import

Docker service can be imported using the long id, e.g. for a service with the short id `55ba873dd`:

```sh
$ terraform import docker_service.foo $(docker service inspect -f {{.ID}} 55b)
```
