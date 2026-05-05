resource "docker_compose" "test" {
  project_name   = "%s"
  remove_orphans = true
  wait           = true
  wait_timeout   = "30s"
  profiles       = ["extras"]
  env_files = [
    "%s/testAccDockerCompose.env",
  ]
  config_paths = [
    "%s/testAccDockerComposeProfiles.compose.yaml",
  ]
}