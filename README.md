# SeatGeek Fork of kreuzwerker/docker

This repo is a SeatGeek fork of the [kreuzwerker/docker](https://github.com/kreuzwerker/terraform-provider-docker), 
the Terraform Docker Provider. It contains a [single PR](https://github.com/seatgeek/terraform-provider-docker/pull/1)
that has been [submitted](https://github.com/kreuzwerker/terraform-provider-docker/pull/565) to the source repo but has
not been looked at. 

The PR adds a `docker_registry_multiarch_image` data source to expose details about
[Docker multi-arch images](https://docs.docker.com/build/building/multi-platform/). When you build a multi-arch image, the
underlying images are not tagged so it is useful to be able to look up details about them. There is a usage example
available [here](examples/data-sources/docker_registry_multiarch_image/data-source.tf).

The long term goal is to have this change accepted into the upstream and deprecate this repo. 

## License

The Terraform Provider Docker is available to everyone under the terms of the Mozilla Public License Version 2.0. [Take a look the LICENSE file](LICENSE).
