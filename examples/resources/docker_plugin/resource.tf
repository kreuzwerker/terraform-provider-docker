resource "docker_plugin" "sample-volume-plugin" {
  name = "docker.io/tiborvass/sample-volume-plugin:latest"
}

resource "docker_plugin" "sample-volume-plugin" {
  name                  = "tiborvass/sample-volume-plugin"
  alias                 = "sample-volume-plugin"
  enabled               = false
  grant_all_permissions = true
  force_destroy         = true
  enable_timeout        = 60
  force_disable         = true
  env = [
    "DEBUG=1"
  ]
}
