provider "docker" {
  alias = "private"
  registry_auth {
    address             = "%s"
    config_file_content = file("%s")
  }
}
resource "docker_image" "foo_private" {
  provider = "docker.private"
  name     = "%s"
}
