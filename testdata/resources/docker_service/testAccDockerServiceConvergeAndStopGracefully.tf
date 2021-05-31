provider "docker" {
  registry_auth {
    address = "127.0.0.1:15000"
  }
}

resource "docker_service" "foo" {
  name = "tftest-service-basic-converge"
  task_spec {
    container_spec {
      image             = "127.0.0.1:15000/tftest-service:v1"
      stop_grace_period = "10s"
      healthcheck {
        test         = ["CMD", "curl", "-f", "localhost:8080/health"]
        interval     = "5s"
        timeout      = "2s"
        start_period = "0s"
        retries      = 4
      }
    }
  }

  mode {
    replicated {
      replicas = 2
    }
  }

  endpoint_spec {
    ports {
      target_port = "8080"
    }
  }

  converge_config {
    delay   = "7s"
    timeout = "3m"
  }
}
