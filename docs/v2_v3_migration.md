# V2 to V3 Migration Guide

This guide is intended to help you migrate from V2 to V3 of the `terraform-provider-docker`.

The in the past minor versions there were many new attributes and older attributes are deprecated.

This will give you an overview over which attributes are deprecated and which attributes you should use instead.

## `docker_container`


Deprecated attributes:

* `links`: The --link flag is a legacy feature of Docker and will be removed (https://docs.docker.com/network/links/)
* `ip_address`, `ip_prefix_length`, `gateway`: Use the `network_data` block instead
* `network_alias`, `networks`: Use the `networks_advanced` block instead


## `docker_image`

* `latest`: Use `repo_digest` instead
* `pull_trigger`: Use `pull_triggers` instead
* `output`: Unused and will be removed
* `build.path`: Use `build.context` instead

## `docker_service`

* `networks`: Use the `networks_advanced` block instead


## `docker_registry_image`

The whole `build` block will be removed. Use the `build` block of the `docker_image` resource instead.
In order to push images to an registry, still use `docker_registry_image` and reference the `docker_image` resource:

```hcl
resource "docker_image" "foo_image" {
  provider = "docker.private"
  name     = "somename"
  build {
    // your build params
  }
}

resource "docker_registry_image" "foo" {
  name = docker_image.foo_image.name
}
```
