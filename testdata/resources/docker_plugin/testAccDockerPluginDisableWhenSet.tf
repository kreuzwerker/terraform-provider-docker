resource "docker_plugin" "test" {
  name                  = "docker.io/tiborvass/sample-volume-plugin:latest"
  alias                 = "sample:latest"
  grant_all_permissions = true
  force_destroy         = true
  enable_timeout        = 60
  env = [
    "DEBUG=1"
  ]
}
