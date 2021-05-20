resource "docker_service" "foo" {
  name = "foo"

  task_spec {
    container_spec {
      image = "nginx"
    }
  }

  endpoint_spec {
    ports {
      target_port    = "80"
      published_port = "8080"

    }
  }
}
