---
layout: "docker"
page_title: "Docker: docker_registry_image"
sidebar_current: "docs-docker-resource-registry-image"
description: |-
  Manages the lifecycle of docker image/tag in a registry.
---

# docker\_registry\_image

Provides an image/tag in a Docker registry.

## Example Usage

```hcl
resource "docker_registry_image" "helloworld" {

  name = "helloworld:1.0"

  build {
    context = "pathToContextFolder"
  }

}

```

## Argument Reference

* `name` - (Required, string) The name of the Docker image.
* `keep_remotely` - (Optional, boolean) If true, then the Docker image won't be
  deleted on destroy operation. If this is false, it will delete the image from
  the docker registry on destroy operation.

* `build` - (Optional, Map) See [Build](#build-1) below for details.

<a id="build-1"></a>
#### Build Block

* `context` (Required, string) - The path to the context folder
* `suppress_output` (Optional, bool) - Suppress the build output and print image ID on success
* `remote_context` (Optional, string) - A Git repository URI or HTTP/HTTPS context URI
* `no_cache` (Optional, bool) - Do not use the cache when building the image
* `remove` (Optional, bool) - Remove intermediate containers after a successful build (default behavior)
* `force_remove` (Optional, bool) - Always remove intermediate containers
* `pull_parent` (Optional, bool) - Attempt to pull the image even if an older image exists locally
* `isolation` (Optional, string) - Isolation represents the isolation technology of a container. The supported values are platform specific
* `cpu_set_cpus` (Optional, string) - CPUs in which to allow execution (e.g., 0-3, 0,1)
* `cpu_set_mems` (Optional, string) - MEMs in which to allow execution (0-3, 0,1)
* `cpu_shares` (Optional, int) - CPU shares (relative weight)
* `cpu_quota` (Optional, int) - Microseconds of CPU time that the container can get in a CPU period
* `cpu_period` (Optional, int) - The length of a CPU period in microseconds
* `memory` (Optional, int) - Set memory limit for build
* `memory_swap` (Optional, int) - Total memory (memory + swap), -1 to enable unlimited swap
* `cgroup_parent` (Optional, string) - Optional parent cgroup for the container
* `network_mode` (Optional, string) - Set the networking mode for the RUN instructions during build
* `shm_size` (Optional, int) - Size of /dev/shm in bytes. The size must be greater than 0
* `` (Optional, string) - Set the networking mode for the RUN instructions during build
* `dockerfile` (Optional, string) - Dockerfile file. Default is "Dockerfile"
* `ulimit` (Optional, Map) - See [Ulimit](#ulimit-1) below for details
* `build_args` (Optional, map of key/value pairs) string pairs for build-time variables
* `auth_config` (Optional, Map) - See [AuthConfig](#authconfig-1) below for details
* `labels` (Optional, map of key/value pairs) string pairs for labels
* `squash` (Optional, bool) - squash the new layers into a new image with a single new layer
* `cache_from` (Optional, []string) - Images to consider as cache sources
* `security_opt` (Optional, []string) - Security options
* `extra_hosts` (Optional, []string) - A list of hostnames/IP mappings to add to the container’s /etc/hosts file. Specified in the form ["hostname:IP"]
* `target` (Optional, string) - Set the target build stage to build
* `platform` (Optional, string) - Set platform if server is multi-platform capable
* `version` (Optional, string) - Version of the unerlying builder to use
* `build_id` (Optional, string) - BuildID is an optional identifier that can be passed together with the build request. The same identifier can be used to gracefully cancel the build with the cancel request

<a id="ulimit-1"></a>
#### Ulimit Block

* `name` - (Required, string) type of ulimit, e.g. nofile
* `soft` (Required, int) - soft limit
* `hard` (Required, int) - hard limit

<a id="authconfig-1"></a>
#### AuthConfig Block

* `host_name` - (Required, string) hostname of the registry
* `user_name` - (Optional, string) the registry user name
* `password` - (Optional, string) the registry password
* `auth` - (Optional, string) the auth token
* `email` - (Optional, string) the user emal
* `server_address` - (Optional, string) the server address
* `identity_token` - (Optional, string) the identity token
* `registry_token` - (Optional, string) the registry token

## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `sha256_digest` (string) - The sha256 digest of the image.
