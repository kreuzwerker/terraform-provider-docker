resource "docker_image" "outside_context" {
  name = "outside-context:latest"

  build {
    path = "."
    dockerfile = "../Dockerfile"
  }
}

