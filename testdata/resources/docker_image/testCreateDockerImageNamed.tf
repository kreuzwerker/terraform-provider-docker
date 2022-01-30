resource "docker_image" "test" {
  name = "%s"
  build {
    path         = "."
    dockerfile   = "Dockerfile"
    force_remove = true
  }
}
