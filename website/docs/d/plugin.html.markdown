---
layout: "docker"
page_title: "Docker: docker_plugin"
sidebar_current: "docs-docker-datasource-plugin"
description: |-
  Reads the local Docker pluign.
---

# docker\_plugin

Reads the local Docker plugin. The plugin must be installed locally.

## Example Usage

```hcl
data "docker_plugin" "sample-volume-plugin" {
  alias = "sample-volume-plugin:latest"
}
```

## Argument Reference

The following arguments are supported:

* `id` - (Optional, string) The Docker plugin ID.
* `alias` - (Optional, string) The alias of the Docker plugin.

One of `id` or `alias` must be assigned.

## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `plugin_reference` - (Optional, string, Forces new resource) The plugin reference.
* `disabled` - (Optional, boolean) If true, the plugin is disabled.
* `grant_all_permissions` - (Optional, boolean) If true, grant all permissions necessary to run the plugin.
* `args` - (Optional, set of string). Currently, only environment variables are supported.
