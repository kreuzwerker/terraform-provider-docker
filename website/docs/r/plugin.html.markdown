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
  args = [
    "DEBUG=1"
  ]
}
```

## Argument Reference

The following arguments are supported:

* `plugin_reference` - (Required, string, Forces new resource) The plugin reference. The registry path and image tag should not be omitted. See [plugin_references, alias](#plugin-references-alias-1) below for details.
* `alias` - (Optional, string, Forces new resource) The alias of the Docker plugin. The image tag should not be omitted. See [plugin_references, alias](#plugin-references-alias-1) below for details.
* `disabled` - (Optional, boolean) If true, the plugin is disabled.
* `grant_all_permissions` - (Optional, boolean) If true, grant all permissions necessary to run the plugin.
* `args` - (Optional, set of string). Currently, only environment variables are supported.
* `disable_when_set` - (Optional, boolean) If true, the plugin becomes disabled temporarily when the plugin setting is updated. See [disable_when_set](#disable-when-set-1) below for details.
* `force_destroy` - (Optional, boolean) If true, the plugin is removed forcely when the plugin is removed.
* `enable_timeout` - (Optional, int) HTTP client timeout to enable the plugin.
* `force_disable` - (Optional, boolean) If true, then the plugin is disabled forcely when the plugin is disabled.

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
      - disabled         = false -> null
      ~ env              = [
          - "DEBUG=0",
        ] -> (known after apply)
      ~ id               = "27784976e1471c1e473a901a0a02055ddc8bc1c9dec9c44d81a49d516c0c28f9" -> (known after apply)
      ~ plugin_reference = "docker.io/tiborvass/sample-volume-plugin:latest" -> "tiborvass/sample-volume-plugin" # forces replacement
    }

Plan: 1 to add, 0 to change, 1 to destroy.
```

<a id="disable-when-set-1"></a>
## disable_when_set

To update the plugin settings, the plugin must be disabled.
Otherwise, it failed to update the plugin settings as the following.

```sh
$ docker plugin set tiborvass/sample-volume-plugin:latest DEBUG=2
Error response from daemon: cannot set on an active plugin, disable plugin before setting
```

If `disable_when_set` is true, then the plugin becomes disabled temporarily before the attribute `args` is updated and after `args` is updated the plugin becomes enabled again.

## Attributes Reference

## Import

Docker plugins can be imported using the long id, e.g. for a plugin `tiborvass/sample-volume-plugin:latest`:

```sh
$ terraform import docker_plugin.sample-volume-plugin $(docker plugin inspect -f "{{.ID}}" tiborvass/sample-volume-plugin:latest)
```
