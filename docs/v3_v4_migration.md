# V3 to V4 Migration Guide

## General

Bump of minimum terraform version to `1.1.5` or newer. This is done as part of introducing the new `terraform-plugin-framework` to develop this provider.

## `docker_image`

**Reworked handling of context and Dockerfile:** This probably is not a breaking change, but more a big bugfix. The build logic now correctly resolves the Dockerfile path for both relative and absolute cases.


## `docker_container`

**Reworked handling of stopped containers:** If a container is stopped (or exists for some other reason), Terraform now correctly shows a change on `plan` and restarts the container on `apply`. To trigger the change, the `must_run` attribute is exploited. `must_run` defaults to `true` and when a container is in a not running state, the provider sets `must_run` to `false` to trigger a state change. This fixes the cases where a stopped container gets deleted during a `plan`. No migration needed.

`ports` - **Ports on stopped container force replacement:** This is now fixed through https://github.com/kreuzwerker/terraform-provider-docker/pull/842, no migration needed.

`devices` - **Fix the replacement of devices** Using `devices` blocks with not all 3 attributes now does not trigger resource replacements anymore. This fixes https://github.com/kreuzwerker/terraform-provider-docker/issues/603.


## `docker_network`

Removed attributes:

* `check_duplicate`

## `docker_service`

New attribute:

* `networks_advanced.id`

Deprecated attribute:

* `networks_advanced.name`: Replaced by `id` attribute to make it clear that the `docker_network.id` needs to be used to prevent constant recreation of the service