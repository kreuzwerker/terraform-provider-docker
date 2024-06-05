resource "docker_image" "test" {
  name = "foo"

  timeouts {
    create = "%s"
  }

  build {
    path         = "."
    dockerfile   = "Dockerfile"
    force_remove = true
  }
}
