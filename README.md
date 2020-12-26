# Terraform Provider

- Website: https://www.terraform.io
- Provider: [kreuzwerker/docker](https://registry.terraform.io/providers/kreuzwerker/docker/latest)
- Slack: [@gophers/terraform-provider-docker](https://gophers.slack.com/archives/C01G9TN5V36)


## Requirements
-	[Terraform](https://www.terraform.io/downloads.html) >=0.12.x
-	[Go](https://golang.org/doc/install) 1.15.x (to build the provider plugin)

## Building The Provider

```sh
$ git clone git@github.com:terraform-providers/terraform-provider-docker
$ make build
```

## Example usage
```hcl
# Set the required provider and versions
terraform {
  required_providers {
    # We recommend pinning to the specific version of the Docker Provider you're using
    # since new versions are released frequently
    docker = {
      source  = "kreuzwerker/docker"
      version = "2.8.0"
    }
  }
}

# Configure the docker provider
provider "docker" {
}

# Create a docker image resource
# -> docker pull nginx:latest
resource "docker_image" "foo" {
  name         = "c"
  keep_locally = true
}

# Create a docker container resource
# -> docker run --name foo -d nginx:latest
resource "docker_container" "foo" {
  name    = "foo"
  image   = docker_image.foo.latest
}
```