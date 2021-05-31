resource "docker_plugin" "test" {
  name                  = "docker.io/vieux/sshfs:latest"
  grant_all_permissions = true
  force_destroy         = true
}
