<a href="https://docker.com">
    <img src="https://raw.githubusercontent.com/kreuzwerker/terraform-provider-docker/master/assets/docker-logo.png" alt="Docker logo" title="Docker" align="right" height="100" />
</a>
<a href="https://terraform.io">
    <img src="https://raw.githubusercontent.com/kreuzwerker/terraform-provider-docker/master/assets/terraform-logo.png" alt="Terraform logo" title="Terraform" align="right" height="100" />
</a>
<a href="https://kreuzwerker.de">
    <img src="https://raw.githubusercontent.com/kreuzwerker/terraform-provider-docker/master/assets/xw-logo.png" alt="Kreuzwerker logo" title="Kreuzwerker" align="right" height="100" />
</a>

# Terraform Provider for Docker

- Website: https://www.terraform.io
- Provider: [kreuzwerker/docker](https://registry.terraform.io/providers/kreuzwerker/docker/latest)
- Slack: [@gophers/terraform-provider-docker](https://gophers.slack.com/archives/C01G9TN5V36)


## Requirements
-	[Terraform](https://www.terraform.io/downloads.html) >=0.12.x
-	[Go](https://golang.org/doc/install) 1.16.x (to build the provider plugin)

## Building The Provider

```sh
$ git clone git@github.com:kreuzwerker/terraform-provider-docker
$ make build
```

## Example usage

Take a look at the examples in the [documentation](https://registry.terraform.io/providers/kreuzwerker/docker/2.13.0/docs) of the registry
or use the following example:


```hcl
# Set the required provider and versions
terraform {
  required_providers {
    # We recommend pinning to the specific version of the Docker Provider you're using
    # since new versions are released frequently
    docker = {
      source  = "kreuzwerker/docker"
      version = "2.13.0"
    }
  }
}

# Configure the docker provider
provider "docker" {
}

# Create a docker image resource
# -> docker pull nginx:latest
resource "docker_image" "nginx" {
  name         = "nginx:latest"
  keep_locally = true
}

# Create a docker container resource
# -> same as 'docker run --name nginx -p8080:80 -d nginx:latest'
resource "docker_container" "nginx" {
  name    = "nginx"
  image   = docker_image.nginx.latest

  ports {
    external = 8080
    internal = 80
  }
}

# Or create a service resource
# -> same as 'docker service create -d -p 8081:80 --name nginx-service --replicas 2 nginx:latest'
resource "docker_service" "nginx_service" {
  name = "nginx-service"
  task_spec {
    container_spec {
      image = docker_image.nginx.repo_digest
    }
  }

  mode {
    replicated {
      replicas = 2
    }
  }

  endpoint_spec {
    ports {
      published_port = 8081
      target_port    = 80
    }
  }
}
```

## Contributing

The Terraform Docker Provider is the work of many of contributors. We appreciate your help!

To contribute, please read the contribution guidelines: [Contributing to Terraform - Docker Provider](CONTRIBUTING.md)

## License

The Terraform Provider Docker is available to everyone under the terms of the Mozilla Public License Version 2.0. [Take a look the LICENSE file](LICENSE).