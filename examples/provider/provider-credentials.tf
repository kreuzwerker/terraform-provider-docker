provider "docker" {
  host = "tcp://localhost:2376"

  registry_auth {
    address     = "index.docker.io"
    config_file = pathexpand("~/.docker/config.json")
  }

  registry_auth {
    address             = "registry.my.company.com"
    config_file_content = var.plain_content_of_config_file
  }

  registry_auth {
    address  = "quay.io:8181"
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
