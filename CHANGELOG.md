## 2.11.0 (January 22, 2021)

IMPROVEMENTS:
- add properties -it (tty and stdin_opn) to docker container ([#120](https://github.com/kreuzwerker/terraform-provider-docker/issues/120))

DOCS
-  fix legacy configuration style ([#126](https://github.com/kreuzwerker/terraform-provider-docker/pull/126))

## 2.10.0 (January 08, 2021)

IMPROVEMENTS:
- add ability to lint/check of links in documentation locally ([#98](https://github.com/kreuzwerker/terraform-provider-docker/pull/98))
- add local semantic commit validation ([#99](https://github.com/kreuzwerker/terraform-provider-docker/pull/99))
- add force_remove option to `r/image` ([#104](https://github.com/kreuzwerker/terraform-provider-docker/pull/104))
- support max replicas of Docker Service Task Spec ([#112](https://github.com/kreuzwerker/terraform-provider-docker/pull/112))
- supports Docker plugin ([#35](https://github.com/kreuzwerker/terraform-provider-docker/pull/35))

DOCS:
- updates docs like gh templates, contribution guideline and readme ([#36](https://github.com/kreuzwerker/terraform-provider-docker/pull/36))
- updates links to provider and slack
- docs(changelog): fixes broken links
- style(changelog): aligns braces
- add labels to arguments of docker_service ([#105](https://github.com/kreuzwerker/terraform-provider-docker/pull/105))

BUG FIXES
- image label for workflows
- set "latest" to tag when tag isn't specified ([#117](https://github.com/kreuzwerker/terraform-provider-docker/pull/117))

CI 
- ci: update ubuntu images and docker version ([#38](https://github.com/kreuzwerker/terraform-provider-docker/pull/38))

## 2.9.0 (December 25, 2020)

IMPROVEMENTS
* Introduces golangci-lint ([#13](https://github.com/kreuzwerker/terraform-provider-docker/issues/13))

BUG FIXES
* docs: devices is a block, not a boolean ([#33](https://github.com/kreuzwerker/terraform-provider-docker/issues/33))
* style: format with gofumpt ([#11](https://github.com/kreuzwerker/terraform-provider-docker/issues/11))
* ci: fix website ci ([#26](https://github.com/kreuzwerker/terraform-provider-docker/issues/26))
* fix: AuxAddress is not read from network and trigger a re-apply every time ([#10](https://github.com/kreuzwerker/terraform-provider-docker/issues/10))

## 2.8.0 (November 11, 2020)

IMPROVEMENTS
* Add new resource docker_registry_image ([#249](https://github.com/terraform-providers/terraform-provider-docker/pull/249))
* Added complete support for Docker credential helpers ([#253](https://github.com/terraform-providers/terraform-provider-docker/pull/253))

BUG FIXES
* Prevent provider error if auth config is incomplete ([#251](https://github.com/terraform-providers/terraform-provider-docker/pull/251))
* Resolve ([#255](https://github.com/terraform-providers/terraform-provider-docker/pull/255) by conditionally adding port binding ([#293](https://github.com/terraform-providers/terraform-provider-docker/pull/293))

DOCS
* Update service.html.markdown ([#245](https://github.com/terraform-providers/terraform-provider-docker/pull/245))
* Documentation updates ([#286](https://github.com/terraform-providers/terraform-provider-docker/pull/286))
* Update link syntax ([#287](https://github.com/terraform-providers/terraform-provider-docker/pull/287))
* Fix typo ([#292](https://github.com/terraform-providers/terraform-provider-docker/pull/292))

CI
* Update to go 1.15 ([#284](https://github.com/terraform-providers/terraform-provider-docker/pull/284))

## 2.7.2 (August 03, 2020)

BUG FIXES
* Fix port objects with the same internal port but different protocol trigger recreation of container ([#274](https://github.com/terraform-providers/terraform-provider-docker/pull/274))
* Fix panic to migrate schema of docker_container from v1 to v2 ([#271](https://github.com/terraform-providers/terraform-provider-docker/pull/271))
* Set `Computed: true` and separate files of resourceDockerContainerV1 ([#272](https://github.com/terraform-providers/terraform-provider-docker/pull/272))
* Prevent force recreate of container about some attributes ([#269](https://github.com/terraform-providers/terraform-provider-docker/pull/269))

DOCS:
* Typo in container.html.markdown ([#278](https://github.com/terraform-providers/terraform-provider-docker/pull/278))
* Update service.html.markdown ([#281](https://github.com/terraform-providers/terraform-provider-docker/pull/281))

## 2.7.1 (June 05, 2020)

BUG FIXES
* prevent force recreate of container about some attributes ([#269](https://github.com/terraform-providers/terraform-provider-docker/issues/269))

## 2.7.0 (February 10, 2020)

IMPROVEMENTS:
* support to import some docker_container's attributes ([#234](https://github.com/terraform-providers/terraform-provider-docker/issues/234))
* make UID, GID, & mode for Docker secrets and configs configurable ([#231](https://github.com/terraform-providers/terraform-provider-docker/pull/231))

BUG FIXES:
* Allow use of `source` file instead of content / content_base64 ([#240](https://github.com/terraform-providers/terraform-provider-docker/pull/240))
* Correct IPAM config read on the data provider ([#229](https://github.com/terraform-providers/terraform-provider-docker/pull/229))
* `published_port` is not correctly populated on docker_service resource ([#222](https://github.com/terraform-providers/terraform-provider-docker/issues/222))
* Registry Config File MUST be a file reference ([#224](https://github.com/terraform-providers/terraform-provider-docker/issues/224))
* Allow zero replicas ([#220](https://github.com/terraform-providers/terraform-provider-docker/pull/220))
* fixing the label schema for HCL2 ([#217](https://github.com/terraform-providers/terraform-provider-docker/pull/217))

DOCS:
* Update documentation to reflect changes in TF v12 ([#228](https://github.com/terraform-providers/terraform-provider-docker/pull/228))

CI:
* bumps docker `19.03` and ubuntu `bionic` ([#241](https://github.com/terraform-providers/terraform-provider-docker/pull/241))

## 2.6.0 (November 25, 2019)

IMPROVEMENTS:
* adds import for resources ([#99](https://github.com/terraform-providers/terraform-provider-docker/issues/99))
* supports --read-only root fs ([#203](https://github.com/terraform-providers/terraform-provider-docker/issues/203))
  
DOCS
* corrects mounts block name in docs ([#218](https://github.com/terraform-providers/terraform-provider-docker/pull/218))


## 2.5.0 (October 15, 2019)

IMPROVEMENTS:
* ci: update to go 1.13 ([#198](https://github.com/terraform-providers/terraform-provider-docker/issues/198))
* feat: migrate to standalone plugin sdk ([#197](https://github.com/terraform-providers/terraform-provider-docker/issues/197))

BUG FIXES:
* fix: removes whitelists of attributes ([#208](https://github.com/terraform-providers/terraform-provider-docker/issues/208))
* fix: splunk Log Driver missing from container `log_driver` ([#204](https://github.com/terraform-providers/terraform-provider-docker/issues/204))


## 2.4.0 (October 07, 2019)

IMPROVEMENTS:
* feat: adds `shm_size attribute` for `docker_container` resource ([#164](https://github.com/terraform-providers/terraform-provider-docker/issues/164))
* feat: supports for group-add ([#191](https://github.com/terraform-providers/terraform-provider-docker/issues/191))

BUG FIXES:
* fix: binary upload as base 64 content ([#48](https://github.com/terraform-providers/terraform-provider-docker/issues/48))
* fix: service env truncation for multiple delimiters ([#121](https://github.com/terraform-providers/terraform-provider-docker/issues/121))
* fix: allows docker_registry_image to read from AWS ECR registry ([#186](https://github.com/terraform-providers/terraform-provider-docker/issues/186))

DOCS
* Removes duplicate `start_period` entry in `healthcheck` section of the documentation for `docker_service` ([#189](https://github.com/terraform-providers/terraform-provider-docker/pull/189))

## 2.3.0 (September 23, 2019)

IMPROVEMENTS:
* feat: adds container ipc mode ([#12](https://github.com/terraform-providers/terraform-provider-docker/issues/12))
* feat: adds container working dir ([#146](https://github.com/terraform-providers/terraform-provider-docker/issues/146))
* remove usage of config pkg ([#183](https://github.com/terraform-providers/terraform-provider-docker/pull/183))

BUG FIXES:
* fix for destroy_grace_seconds is not adhered ([#174](https://github.com/terraform-providers/terraform-provider-docker/issues/174))

## 2.2.0 (August 22, 2019)

IMPROVEMENTS
* Docker client negotiates the version with the server instead of using a fixed version ([#173](https://github.com/terraform-providers/terraform-provider-docker/issues/173))

DOCS
* Fixes section links so they point to the right id ([#176](https://github.com/terraform-providers/terraform-provider-docker/issues/176))

## 2.1.1 (August 08, 2019)

BUG FIXES
* Fixes 'No changes' for containers when all port blocks have been removed ([#167](https://github.com/terraform-providers/terraform-provider-docker/issues/167))

## 2.1.0 (July 19, 2019)

IMPROVEMENTS
* Adds cross-platform support for generic Docker credential helper ([#159](https://github.com/terraform-providers/terraform-provider-docker/pull/159))

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
there are no changes. This was caused to the ascending order. See ([#110](https://github.com/terraform-providers/terraform-provider-docker/issues/110))
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
* Adds labels for `network`, `volume` and `secret` to support docker stacks. ([#92](https://github.com/terraform-providers/terraform-provider-docker/pull/92))
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
