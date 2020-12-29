---
layout: "docker"
page_title: "Docker: docker_plugin"
sidebar_current: "docs-docker-resource-plugin"
description: |-
  Manages the lifecycle of a Docker plugin.
---

# docker\_plugin

Manages the lifecycle of a Docker plugin.

## Example Usage

```hcl
resource "docker_plugin" "sample-volume-plugin" {
  plugin_reference = "docker.io/tiborvass/sample-volume-plugin:latest"
}
```

```hcl
resource "docker_plugin" "sample-volume-plugin" {
  plugin_reference      = "docker.io/tiborvass/sample-volume-plugin:latest"
  alias                 = "sample-volume-plugin:latest"
  disabled              = true
  grant_all_permissions = true
  disable_when_set      = true
  force_destroy         = true
  enable_timeout        = 60
  force_disable         = true
  env = [
    "DEBUG=1"
  ]
}
```

## Argument Reference

The following arguments are supported:

* `plugin_reference` - (Required, string, Forces new resource) The plugin reference. The registry path and image tag should not be omitted.
* `alias` - (Optional, string, Forces new resource) The alias of the Docker plugin. The image tag should not be omitted.
* `disabled` - (Optional, boolean) If true, the plugin is disabled.
* `grant_all_permissions` - (Optional, boolean) If true, grant all permissions necessary to run the plugin.
* `disable_when_set` - (Optional, boolean) If true, the plugin becomes disabled temporarily when the plugin setting is updated.
* `force_destroy` - (Optional, boolean) If true, the plugin becomes disabled temporarily when the plugin setting is updated.
* `env` - (Optional, set of string)
* `enable_timeout` - (Optional, int) HTTP client timeout to enable the plugin.
* `force_disable` - (Optional, boolean) If true, then the plugin is disabled forcely when the plugin is disabled.

## Attributes Reference

## Import

Docker plugins can be imported using the long id, e.g. for a plugin `tiborvass/sample-volume-plugin:latest`:

```sh
$ terraform import docker_plugin.sample-volume-plugin $(docker plugin inspect -f "{{.ID}}" tiborvass/sample-volume-plugin:latest)
```
