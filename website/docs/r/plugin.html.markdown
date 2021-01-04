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

* `plugin_reference` - (Required, string, Forces new resource) The plugin reference. The registry path and image tag should not be omitted. See [plugin_references, alias](#plugin-references-alias-1) below for details.
* `alias` - (Optional, string, Forces new resource) The alias of the Docker plugin. The image tag should not be omitted. See [plugin_references, alias](#plugin-references-alias-1) below for details.
* `enabled` - (Optional, boolean) If true, the plugin is enabled. The default value is `true`.
* `grant_all_permissions` - (Optional, boolean) If true, grant all permissions necessary to run the plugin. This attribute conflicts with `grant_permissions`.
* `grant_permissions` - (Optional, block) grant permissions necessary to run the plugin. This attribute conflicts with `grant_all_permissions`. See [grant_permissions](#grant-permissions-1) below for details.
* `env` - (Optional, set of string). The environment variables.
* `force_destroy` - (Optional, boolean) If true, the plugin is removed forcibly when the plugin is removed.
* `enable_timeout` - (Optional, int) HTTP client timeout to enable the plugin.
* `force_disable` - (Optional, boolean) If true, then the plugin is disabled forcibly when the plugin is disabled.

<a id="plugin-references-alias-1"></a>
## plugin_reference, alias

`plugin_reference` and `alias` must be full path. Otherwise, after `terraform apply` is run, there would be diffs of them.

For example,

```hcl
resource "docker_plugin" "sample-volume-plugin" {
  plugin_reference = "tiborvass/sample-volume-plugin" # must be "docker.io/tiborvass/sample-volume-plugin:latest"
  alias            = "sample"                         # must be "sample:latest"
}
```

```sh
$ terraform apply # a plugin is installed

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.

$ terraform plan

An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
-/+ destroy and then create replacement

Terraform will perform the following actions:

  # docker_plugin.sample-volume-plugin must be replaced
-/+ resource "docker_plugin" "sample-volume-plugin" {
      ~ alias            = "sample:latest" -> "sample" # forces replacement
      - enabled          = false -> null
      ~ env              = [
          - "DEBUG=0",
        ] -> (known after apply)
      ~ id               = "27784976e1471c1e473a901a0a02055ddc8bc1c9dec9c44d81a49d516c0c28f9" -> (known after apply)
      ~ plugin_reference = "docker.io/tiborvass/sample-volume-plugin:latest" -> "tiborvass/sample-volume-plugin" # forces replacement
    }

Plan: 1 to add, 0 to change, 1 to destroy.
```

<a id="grant-permissions-1"></a>
## grant_permissions

`grant_permissions` is a block within the configuration that can be repeated to grant permissions to install the plugin. Each `grant_permissions` block supports
the following:

* `name` - (Required, string)
* `value` - (Required, list of string)

Example:

```hcl
resource "docker_plugin" "sshfs" {
  plugin_reference = "docker.io/vieux/sshfs:latest"
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

## Import

Docker plugins can be imported using the long id, e.g. for a plugin `tiborvass/sample-volume-plugin:latest`:

```sh
$ terraform import docker_plugin.sample-volume-plugin $(docker plugin inspect -f "{{.ID}}" tiborvass/sample-volume-plugin:latest)
```
