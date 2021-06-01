resource "docker_image" "foo" {
  name         = "repo.mycompany.com:8080/foo-service:v1"
  keep_locally = true
}

resource "docker_service" "foo" {
  name = "foo-service"

  task_spec {
    container_spec {
      image = docker_image.foo.latest
    }
  }

  endpoint_spec {
    ports {
      target_port = "8080"
    }
  }
}
