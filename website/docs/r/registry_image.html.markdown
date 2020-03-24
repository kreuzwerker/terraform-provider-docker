---
layout: "docker"
page_title: "Docker: docker_registry_image"
sidebar_current: "docs-docker-resource-registry-image"
description: |-
  Manages the lifecycle of docker image/tag in a registry.
---

# docker\_registry\_image

Provides an image/tag in a Docker registry.

## Example Usage

```hcl
resource "docker_registry_image" "helloworld" {

  name = "helloworld:1.0"

  build {
    context = "pathToContextFolder"
  }

}

```

## Argument Reference

* `name` - (Required, string) The name of the Docker image.
* `keep_remotely` - (Optional, boolean) If true, then the Docker image won't be
  deleted on destroy operation. If this is false, it will delete the image from
  the docker registry on destroy operation.

* `build` - (Optional, Customize docker build arguments) See [Build](#build-1) below for details.

<a id="build-1"></a>
#### Build Block



## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `sha256_digest` (string) - The sha256 digest of the image.
