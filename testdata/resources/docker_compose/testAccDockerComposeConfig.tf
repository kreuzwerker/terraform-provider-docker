resource "docker_compose" "test" {
  project_name   = "%s"
  remove_orphans = true
  config_paths = [
    "%s/testAccDockerComposeConfig.compose.yaml",
  ]
}