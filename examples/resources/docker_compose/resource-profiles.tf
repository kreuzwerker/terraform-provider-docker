resource "docker_compose" "app" {
  project_name   = "example-compose-app"
  remove_orphans = true
  wait           = true
  wait_timeout   = "30s"
  profiles       = ["frontend"]
  env_files = [
    "${path.module}/app.env",
  ]
  config_paths = [
    "${path.module}/compose.yaml",
    "${path.module}/compose.override.yaml",
  ]
}