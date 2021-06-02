provider "docker" {
  alias = "private"
  registry_auth {
    address = "127.0.0.1:15000"
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
  name     = "tftest-service-updateFailsAndRollbackConverge"
  task_spec {
    container_spec {
      image             = docker_image.tftest_image.latest
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

  update_config {
    parallelism       = 1
    delay             = "5s"
    failure_action    = "rollback"
    monitor           = "10s"
    max_failure_ratio = "0.0"
    order             = "stop-first"
  }

  rollback_config {
    parallelism       = 1
    delay             = "1s"
    failure_action    = "pause"
    monitor           = "4s"
    max_failure_ratio = "0.0"
    order             = "stop-first"
  }

  endpoint_spec {
    mode = "vip"
    ports {
      name           = "random"
      protocol       = "tcp"
      target_port    = "8080"
      published_port = "8080"
      publish_mode   = "ingress"
    }
  }

  converge_config {
    delay   = "7s"
    timeout = "3m"
  }
}
