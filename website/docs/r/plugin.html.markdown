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
  name  = "tiborvass/sample-volume-plugin:latest"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the Docker plugin.
* `disabled` - (Optional, boolean) If true, the plugin is disabled.

## Attributes Reference

## Import

Docker plugins can be imported using the long id, e.g. for a plugin `tiborvass/sample-volume-plugin:latest`:

```
$ terraform import docker_plugin.sample-volume-plugin $(docker plugin inspect -f "{{.ID}}" tiborvass/sample-volume-plugin:latest)
```
