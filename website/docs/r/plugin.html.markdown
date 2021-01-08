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
  name = "docker.io/tiborvass/sample-volume-plugin:latest"
}
```

```hcl
resource "docker_plugin" "sample-volume-plugin" {
  name                  = "tiborvass/sample-volume-plugin"
  alias                 = "sample-volume-plugin"
  enabled               = false
  grant_all_permissions = true
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

* `name` - (Required, string, Forces new resource) The plugin name. If the tag is omitted, `:latest` is complemented to the attribute value.
* `alias` - (Optional, string, Forces new resource) The alias of the Docker plugin. If the tag is omitted, `:latest` is complemented to the attribute value.
* `enabled` - (Optional, boolean) If true, the plugin is enabled. The default value is `true`.
* `grant_all_permissions` - (Optional, boolean) If true, grant all permissions necessary to run the plugin. This attribute conflicts with `grant_permissions`.
* `grant_permissions` - (Optional, block) grant permissions necessary to run the plugin. This attribute conflicts with `grant_all_permissions`. See [grant_permissions](#grant-permissions-1) below for details.
* `env` - (Optional, set of string). The environment variables.
* `force_destroy` - (Optional, boolean) If true, the plugin is removed forcibly when the plugin is removed.
* `enable_timeout` - (Optional, int) HTTP client timeout to enable the plugin.
* `force_disable` - (Optional, boolean) If true, then the plugin is disabled forcibly when the plugin is disabled.

<a id="grant-permissions-1"></a>
## grant_permissions

`grant_permissions` is a block within the configuration that can be repeated to grant permissions to install the plugin. Each `grant_permissions` block supports
the following:

* `name` - (Required, string)
* `value` - (Required, list of string)

Example:

```hcl
resource "docker_plugin" "sshfs" {
  name = "docker.io/vieux/sshfs:latest"
  grant_permissions {
    name = "network"
    value = [
      "host"
    ]
  }
  grant_permissions {
    name = "mount"
    value = [
      "",
      "/var/lib/docker/plugins/"
    ]
  }
  grant_permissions {
    name = "device"
    value = [
      "/dev/fuse"
    ]
  }
  grant_permissions {
    name = "capabilities"
    value = [
      "CAP_SYS_ADMIN"
    ]
  }
}
```

## Attributes Reference

* `plugin_reference` - (string) The plugin reference.

## Import

Docker plugins can be imported using the long id, e.g. for a plugin `tiborvass/sample-volume-plugin:latest`:

```sh
$ terraform import docker_plugin.sample-volume-plugin $(docker plugin inspect -f "{{.ID}}" tiborvass/sample-volume-plugin:latest)
```
