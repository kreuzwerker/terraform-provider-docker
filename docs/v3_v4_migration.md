# V3 to V4 Migration Guide

## General

Bump of minimum terraform version to `1.1.5` or newer. This is done as part of introducing the new `terraform-plugin-framework` to develop this provider.


## `docker_network`

Removed attributes:

* `check_duplicate`

## `docker_service`

New attribute:

* `networks_advanced.id`

Deprecated attribute:

* `networks_advanced.name`: Replaced by `id` attribute to make it clear that the `docker_network.id` needs to be used to prevent constant recreation of the service