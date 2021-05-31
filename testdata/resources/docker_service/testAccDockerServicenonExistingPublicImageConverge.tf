resource "docker_service" "foo" {
  name = "tftest-service-publicimagedoesnotexist"
  task_spec {
    container_spec {
      image = "stovogel/blablabla:part5"
    }
  }

  mode {
    replicated {
      replicas = 2
    }
  }

  converge_config {
    delay   = "7s"
    timeout = "10s"
  }
}
