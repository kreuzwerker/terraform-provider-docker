---
layout: "docker"
page_title: "Provider: Docker"
sidebar_current: "docs-docker-index"
description: |-
  The Docker provider is used to interact with Docker containers and images.
---

# Docker Provider

The Docker provider is used to interact with Docker containers and images.
It uses the Docker API to manage the lifecycle of Docker containers. Because
the Docker provider uses the Docker API, it is immediately compatible not
only with single server Docker but Swarm and any additional Docker-compatible
API hosts.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Docker provider
provider "docker" {
  host = "tcp://127.0.0.1:2376/"
}

# Create a container
resource "docker_container" "foo" {
  image = docker_image.ubuntu.latest
  name  = "foo"
}

resource "docker_image" "ubuntu" {
  name = "ubuntu:latest"
}
```

-> **Note**
You can also use the `ssh` protocol to connect to the docker host on a remote machine.
The configuration would look as follows:

```hcl
provider "docker" {
  host = "ssh://user@remote-host:22"
}
```

## Registry Credentials

Registry credentials can be provided on a per-registry basis with the `registry_auth`
field, passing either a config file or the username/password directly.

-> **Note**
The location of the config file is on the machine terraform runs on, nevertheless if the specified docker host is on another machine.

``` hcl
provider "docker" {
  host = "tcp://localhost:2376"

  registry_auth {
    address = "registry.hub.docker.com"
    config_file = pathexpand("~/.docker/config.json")
  }

  registry_auth {
    address = "registry.my.company.com"
    config_file_content = var.plain_content_of_config_file
  }

  registry_auth {
    address = "quay.io:8181"
    username = "someuser"
    password = "somepass"
  }
}

data "docker_registry_image" "quay" {
  name = "myorg/privateimage"
}

data "docker_registry_image" "quay" {
  name = "quay.io:8181/myorg/privateimage"
}
```

-> **Note**
When passing in a config file either the corresponding `auth` string of the repository is read or the os specific
credential helpers (see [here](https://github.com/docker/docker-credential-helpers#available-programs)) are
used to retrieve the authentication credentials.

You can still use the environment variables `DOCKER_REGISTRY_USER` and `DOCKER_REGISTRY_PASS`.

An example content of the file `~/.docker/config.json` on macOS may look like follows:

```json
{
	"auths": {
		"repo.mycompany:8181": {
			"auth": "dXNlcjpwYXNz="
		},
		"otherrepo.other-company:8181": {

		}
	},
  "credsStore" : "osxkeychain"
}
```

## Certificate information

Specify certificate information either with a directory or
directly with the content of the files for connecting to the Docker host via TLS.

```hcl
provider "docker" {
  host = "tcp://your-host-ip:2376/"

  # -> specify either
  cert_path = pathexpand("~/.docker")

  # -> or the following
  ca_material   = file(pathexpand("~/.docker/ca.pem")) # this can be omitted
  cert_material = file(pathexpand("~/.docker/cert.pem"))
  key_material  = file(pathexpand("~/.docker/key.pem"))
}
```

## Argument Reference

The following arguments are supported:

* `host` - (Required) This is the address to the Docker host. If this is
  blank, the `DOCKER_HOST` environment variable will also be read.

* `cert_path` - (Optional) Path to a directory with certificate information
  for connecting to the Docker host via TLS. It is expected that the 3 files `{ca, cert, key}.pem` 
  are present in the path. If the path is blank, the `DOCKER_CERT_PATH` will also be checked.

* `ca_material`, `cert_material`, `key_material`, - (Optional) Content of `ca.pem`, `cert.pem`, and `key.pem` files
  for TLS authentication. Cannot be used together with `cert_path`. If `ca_material` is omitted
  the client does not check the servers certificate chain and host name.

* `registry_auth` - (Optional) A block specifying the credentials for a target
  v2 Docker registry.
   
  * `address` - (Required) The address of the registry.
 
  * `username` - (Optional) The username to use for authenticating to the registry.
  Cannot be used with the `config_file` option. If this is blank, the `DOCKER_REGISTRY_USER`
  will also be checked.
 
  * `password` - (Optional) The password to use for authenticating to the registry.
  Cannot be used with the `config_file` option. If this is blank, the `DOCKER_REGISTRY_PASS`
  will also be checked.
 
  * `config_file` - (Optional) The path to a config file containing credentials for
  authenticating to the registry. Cannot be used with the `username`/`password` or `config_file_content` options.
  If this is blank, the `DOCKER_CONFIG` will also be checked.
  
  * `config_file_content` - (Optional) The content of a config file as string containing credentials for
  authenticating to the registry. Cannot be used with the `username`/`password` or `config_file` options.
 
 

~> **NOTE on Certificates and `docker-machine`:**  As per [Docker Remote API
documentation](https://docs.docker.com/engine/reference/api/docker_remote_api/),
in any docker-machine environment, the Docker daemon uses an encrypted TCP
socket (TLS) and requires `cert_path` for a successful connection. As an alternative,
if using `docker-machine`, run `eval $(docker-machine env <machine-name>)` prior
to running Terraform, and the host and certificate path will be extracted from
the environment.
