provider "docker" {
  registry_auth {
    address = "127.0.0.1:15000"
  }
}

resource "docker_service" "foo" {
  name = "tftest-service-basic"
  task_spec {
    container_spec {
      image = "127.0.0.1:15000/tftest-service:v1"
    }
  }
  mode {
    global = true
  }
  converge_config {
    delay   = "7s"
    timeout = "10s"
  }
}
