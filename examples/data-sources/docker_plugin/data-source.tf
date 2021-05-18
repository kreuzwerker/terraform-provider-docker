### With alias
data "docker_plugin" "by_alias" {
  alias = "sample-volume-plugin:latest"
}

### With ID
data "docker_plugin" "by_id" {
  id = "e9a9db917b3bfd6706b5d3a66d4bceb9f"
}