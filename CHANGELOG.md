## 1.0.0 (July 03, 2018)

NOTES:
* Update `go-dockerclient` to `bf3bc17bb` [#46](https://github.com/terraform-providers/terraform-provider-docker/pull/46)
* The `links` property on `resource_docker_container` is now marked as deprecated [#47](https://github.com/terraform-providers/terraform-provider-docker/pull/47)

FEATURES:
* Add `swarm` capabilities ([#29](https://github.com/terraform-providers/terraform-provider-docker/issues/29)] [#40](https://github.com/terraform-providers/terraform-provider-docker/pull/40) with fixes [#66](https://github.com/terraform-providers/terraform-provider-docker/pull/66) up to Docker `18.03.1` and API Version `1.37` [[#64](https://github.com/terraform-providers/terraform-provider-docker/issues/64))
* Add ability to upload executable files [#55](https://github.com/terraform-providers/terraform-provider-docker/pull/55)
* Add support to attach devices to containers [[#30](https://github.com/terraform-providers/terraform-provider-docker/issues/30)] [#54](https://github.com/terraform-providers/terraform-provider-docker/pull/54)
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
