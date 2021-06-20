data "docker_image" "foo" {
  name = "repo.mycompany.com:8080/foo-service:v1"
}

resource "docker_service" "foo" {
  name = "foo-service"

  task_spec {
    container_spec {
      image = data.docker_image.foo.repo_digest
    }
  }

  endpoint_spec {
    ports {
      target_port = "8080"
    }
  }
}
