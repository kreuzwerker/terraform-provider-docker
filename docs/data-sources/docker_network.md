---
layout: "docker"
page_title: "Docker: docker_network"
sidebar_current: "docs-docker-datasource-docker-network"
description: |-
  `docker_network` provides details about a specific Docker Network.
---

# docker\_network

Finds a specific docker network and returns information about it.

## Example Usage

```hcl
data "docker_network" "main" {
  name = "main"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional, string) The name of the Docker network.
* `id` - (Optional, string) The id of the Docker network.

## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `driver` - (Optional, string) The driver of the Docker network. 
	Possible values are `bridge`, `host`, `overlay`, `macvlan`. 
	See [docker docs][networkdocs] for more details.
* `options` - (Optional, map) Only available with bridge networks. See
	[docker docs][bridgeoptionsdocs] for more details.
* `internal` (Optional, bool) Boolean flag for whether the network is internal.
* `ipam_config` (Optional, map) See [IPAM](#ipam) below for details.
* `scope` (Optional, string) Scope of the network. One of `swarm`, `global`, or `local`.

[networkdocs] https://docs.docker.com/network/#network-drivers
[bridgeoptionsdocs] https://docs.docker.com/engine/reference/commandline/network_create/#bridge-driver-options