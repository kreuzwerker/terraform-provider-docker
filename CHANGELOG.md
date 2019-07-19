## 2.1.1 (Unreleased)
## 2.1.0 (July 19, 2019)

IMPROVEMENTS
* Adds cross-platform support for generic Docker credential helper([#159](https://github.com/terraform-providers/terraform-provider-docker/pull/159))

DOC
* Updates the docs for ssh protocol and mounts ([#158](https://github.com/terraform-providers/terraform-provider-docker/issues/158))
* Fixes website typo / containers / mount vs mounts ([#162](https://github.com/terraform-providers/terraform-provider-docker/pull/162))

## 2.0.0 (June 25, 2019)

BREAKING CHANGES
* Updates to Terraform `v0.12` [[#144](https://github.com/terraform-providers/terraform-provider-docker/issues/144)] and ([#150](https://github.com/terraform-providers/terraform-provider-docker/pull/150))
  
IMPROVEMENTS
* Refactors test setup ([#156](https://github.com/terraform-providers/terraform-provider-docker/pull/156))
* Fixes flaky acceptance tests ([#154](https://github.com/terraform-providers/terraform-provider-docker/pull/154))

## 1.2.0 (May 29, 2019)

IMPROVEMENTS
* Updates to docker `18.09` and API Version `1.39` ([#114](https://github.com/terraform-providers/terraform-provider-docker/issues/114))
* Upgrades to go `1.11` ([#116](https://github.com/terraform-providers/terraform-provider-docker/pull/116))
* Switches to `go modules` ([#124](https://github.com/terraform-providers/terraform-provider-docker/issues/124))
* Adds data source for networks ([#84](https://github.com/terraform-providers/terraform-provider-docker/issues/84))
* Adds `ssh` protocol support ([#153](https://github.com/terraform-providers/terraform-provider-docker/issues/153))
* Adds docker container mounts support ([#147](https://github.com/terraform-providers/terraform-provider-docker/pull/147))

BUG FIXES
* Fixes image pulling and local registry connections ([#143](https://github.com/terraform-providers/terraform-provider-docker/pull/143))

## 1.1.1 (March 08, 2019)

BUG FIXES
* Fixes no more 'force new resource' for container ports when
there are no changes. This was caused to the ascending order. See [[#110](https://github.com/terraform-providers/terraform-provider-docker/issues/110)] 
for details and ([#115](https://github.com/terraform-providers/terraform-provider-docker/pull/115))
* Normalize blank port IP's to 0.0.0.0 ([#128](https://github.com/terraform-providers/terraform-provider-docker/pull/128))

BUILD
* Simplify Dockerfile(s) for tests ([#135](https://github.com/terraform-providers/terraform-provider-docker/pull/135))
* Skip test if swap limit isn't available ([#136](https://github.com/terraform-providers/terraform-provider-docker/pull/136))

DOCS
* Corrects `networks_advanced` section ([#109](https://github.com/terraform-providers/terraform-provider-docker/issues/109))
* Corrects `tmpfs_options` section ([#122](https://github.com/terraform-providers/terraform-provider-docker/issues/122))
* Corrects indentation for container in docs ([#126](https://github.com/terraform-providers/terraform-provider-docker/issues/126))
* Fix syntax error in docker_service example and make all examples adhere to terraform fmt ([#137](https://github.com/terraform-providers/terraform-provider-docker/pull/137))

## 1.1.0 (October 30, 2018)

IMPROVEMENTS
* Adds labels for `network`, `volume` and `secret` to support docker stacks. [[#92](https://github.com/terraform-providers/terraform-provider-docker/pull/92)] 
* Adds `rm` and `attach` options to execute short-lived containers ([#43](https://github.com/terraform-providers/terraform-provider-docker/issues/43)] and [[#106](https://github.com/terraform-providers/terraform-provider-docker/pull/106))
* Adds container healthcheck([#93](https://github.com/terraform-providers/terraform-provider-docker/pull/93))
* Adds the docker container start flag ([#62](https://github.com/terraform-providers/terraform-provider-docker/issues/62)] and [[#94](https://github.com/terraform-providers/terraform-provider-docker/pull/94))
* Adds `cpu_set` to docker container ([#41](https://github.com/terraform-providers/terraform-provider-docker/pull/41))
* Simplifies the image options parser and adds missing registry combinations ([#49](https://github.com/terraform-providers/terraform-provider-docker/pull/49))
* Adds container static IPv4/IPv6 address. Marks network and network_alias as deprecated. ([#105](https://github.com/terraform-providers/terraform-provider-docker/pull/105))
* Adds container logs option ([#108](https://github.com/terraform-providers/terraform-provider-docker/pull/108))

BUG FIXES
* Fixes that new network were appended to the default bridge ([#10](https://github.com/terraform-providers/terraform-provider-docker/issues/10))
* Fixes that container resource returns a non-existent IP address ([#36](https://github.com/terraform-providers/terraform-provider-docker/issues/36))
* Fixes container's ip_address is empty when using custom network ([#9](https://github.com/terraform-providers/terraform-provider-docker/issues/9)] and [[#50](https://github.com/terraform-providers/terraform-provider-docker/pull/50))
* Fixes terraform destroy failing to remove a bridge network ([#98](https://github.com/terraform-providers/terraform-provider-docker/issues/98)] and [[#50](https://github.com/terraform-providers/terraform-provider-docker/pull/50))


## 1.0.4 (October 17, 2018)

BUG FIXES
* Support and fix for random external ports for containers [[#102](https://github.com/terraform-providers/terraform-provider-docker/issues/102)] and ([#103](https://github.com/terraform-providers/terraform-provider-docker/pull/103))

## 1.0.3 (October 12, 2018)

IMPROVEMENTS
* Add support for running tests on Windows [[#54](https://github.com/terraform-providers/terraform-provider-docker/issues/54)] and ([#90](https://github.com/terraform-providers/terraform-provider-docker/pull/90))
* Add options for PID and user namespace mode [[#88](https://github.com/terraform-providers/terraform-provider-docker/issues/88)] and ([#96](https://github.com/terraform-providers/terraform-provider-docker/pull/96))

BUG FIXES
* Fixes issue with internal and external ports on containers [[#8](https://github.com/terraform-providers/terraform-provider-docker/issues/8)] and ([#89](https://github.com/terraform-providers/terraform-provider-docker/pull/89))
* Fixes `tfstate` having correct external port for containers [[#73](https://github.com/terraform-providers/terraform-provider-docker/issues/73)] and ([#95](https://github.com/terraform-providers/terraform-provider-docker/pull/95))
* Fixes that a `docker_image` can be pulled with its SHA256 tag/repo digest [[#79](https://github.com/terraform-providers/terraform-provider-docker/issues/79)] and ([#97](https://github.com/terraform-providers/terraform-provider-docker/pull/97))

## 1.0.2 (September 27, 2018)

BUG FIXES
* Fixes connection via TLS to docker host with file contents ([#86](https://github.com/terraform-providers/terraform-provider-docker/issues/86))
* Skips TLS verification if `ca_material` is not set ([#14](https://github.com/terraform-providers/terraform-provider-docker/issues/14))

## 1.0.1 (August 06, 2018)

BUG FIXES
* Fixes empty strings on mapping from map to slice causes ([#81](https://github.com/terraform-providers/terraform-provider-docker/issues/81))

## 1.0.0 (July 03, 2018)

NOTES:
* Update `go-dockerclient` to `bf3bc17bb` ([#46](https://github.com/terraform-providers/terraform-provider-docker/pull/46))
* The `links` property on `resource_docker_container` is now marked as deprecated ([#47](https://github.com/terraform-providers/terraform-provider-docker/pull/47))

FEATURES:
* Add `swarm` capabilities ([#29](https://github.com/terraform-providers/terraform-provider-docker/issues/29), [#40](https://github.com/terraform-providers/terraform-provider-docker/pull/40) which fixes [#66](https://github.com/terraform-providers/terraform-provider-docker/pull/66) up to Docker `18.03.1` and API Version `1.37` ([#64](https://github.com/terraform-providers/terraform-provider-docker/issues/64))
* Add ability to upload executable files [#55](https://github.com/terraform-providers/terraform-provider-docker/pull/55)
* Add support to attach devices to containers [#30](https://github.com/terraform-providers/terraform-provider-docker/issues/30), [#54](https://github.com/terraform-providers/terraform-provider-docker/pull/54)
* Add Ulimits to containers [#35](https://github.com/terraform-providers/terraform-provider-docker/pull/35)

IMPROVEMENTS:
* Fix `travis` build with a fixed docker version [#57](https://github.com/terraform-providers/terraform-provider-docker/pull/57)
* Infrastructure for Acceptance tests [#39](https://github.com/terraform-providers/terraform-provider-docker/pull/39)
* Internal refactorings [#38](https://github.com/terraform-providers/terraform-provider-docker/pull/38)
* Allow the awslogs log driver [#28](https://github.com/terraform-providers/terraform-provider-docker/pull/28)
* Add prefix `library` only to official images in the path [#27](https://github.com/terraform-providers/terraform-provider-docker/pull/27)

BUG FIXES
* Update documentation for private registries ([#45](https://github.com/terraform-providers/terraform-provider-docker/issues/45))

## 0.1.1 (November 21, 2017)

FEATURES:
* Support for pulling images from private registries [#21](https://github.com/terraform-providers/terraform-provider-docker/issues/21)

## 0.1.0 (June 20, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
