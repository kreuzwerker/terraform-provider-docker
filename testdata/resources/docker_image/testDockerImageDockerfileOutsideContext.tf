resource "docker_image" "outside_context" {
  name = "outside-context:latest"

  build {
    context    = "."
    dockerfile = "../Dockerfile"
  }
}

