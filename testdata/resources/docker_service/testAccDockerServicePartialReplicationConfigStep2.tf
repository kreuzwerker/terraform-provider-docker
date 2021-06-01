provider "docker" {
  registry_auth {
    address = "127.0.0.1:15000"
  }
}

resource "docker_image" "tftest_image" {
  name         = "127.0.0.1:15000/tftest-service:v1"
  keep_locally = false
}

resource "docker_service" "foo" {
  name = "tftest-service-basic"
  task_spec {
    container_spec {
      image             = docker_image.tftest_image.latest
      stop_grace_period = "10s"
    }
  }
  mode {
    replicated {}
  }
}
