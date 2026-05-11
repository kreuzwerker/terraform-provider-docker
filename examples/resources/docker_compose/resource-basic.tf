resource "docker_compose" "app" {
  project_name = "example-compose-app"

  config_paths = [
    "${path.module}/compose.yaml",
  ]
}