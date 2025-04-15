resource "docker_image" "test" {
  name = "foo"

  timeouts {
    create = "%s"
  }

  build {
    context         = "."
    dockerfile   = "Dockerfile"
    force_remove = true
  }
}