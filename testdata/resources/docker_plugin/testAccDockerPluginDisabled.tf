resource "docker_plugin" "test" {
  name                  = "docker.io/tiborvass/sample-volume-plugin:latest"
  alias                 = "sample:latest"
  enabled               = false
  grant_all_permissions = true
  force_destroy         = true
  force_disable         = true
  enable_timeout        = 60
  env = [
    "DEBUG=1"
  ]
}
