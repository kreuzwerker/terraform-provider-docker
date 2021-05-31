provider "docker" {
  registry_auth {
    address = "%s"
  }
}

resource "docker_service" "foo" {
  name = "tftest-service-foo"
  task_spec {
    container_spec {
      image             = "%s"
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
