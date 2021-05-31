resource "docker_service" "foo" {
  name = "tftest-service-privateimagedoesnotexist"
  task_spec {
    container_spec {
      image = "127.0.0.1:15000/idonoexist:latest"
    }
  }

  mode {
    replicated {
      replicas = 2
    }
  }

  converge_config {
    delay   = "7s"
    timeout = "20s"
  }
}
