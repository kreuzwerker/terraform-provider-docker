provider "docker" {
  alias = "private"
  registry_auth {
    address = "%s"
  }
}

data "docker_registry_image" "tftest_image" {
  provider             = "docker.private"
  name                 = "%s"
  insecure_skip_verify = true
}
resource "docker_image" "tftest_image" {
  provider      = "docker.private"
  name          = data.docker_registry_image.tftest_image.name
  keep_locally  = false
  force_remove  = true
  pull_triggers = [data.docker_registry_image.tftest_image.sha256_digest]
}

resource "docker_service" "foo" {
  provider = "docker.private"
  name     = "tftest-service-foo"
  task_spec {
    container_spec {
      image             = docker_image.tftest_image.latest
      stop_grace_period = "10s"

    }
  }
  mode {
    replicated {
      replicas = 2
    }
  }

  converge_config {
    delay   = "7s"
    timeout = "3m"
  }
}
