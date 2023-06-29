provider "docker" {
    alias = "private"
    registry_auth {
      address       = "%s"
      auth_disabled = true
    }
}
data "docker_registry_image" "foobar" {
    provider             = "docker.private"
    name                 = "127.0.0.1:15002/tftest-service:v1"
    insecure_skip_verify = true
}