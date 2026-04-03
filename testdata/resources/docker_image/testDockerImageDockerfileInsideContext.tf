resource "docker_image" "backend" {
  name = "empty:latest"
  build {
    context = "%s"
    dockerfile = "%s"
    builder = "default"
  }

  triggers = {
    # Change this value to manually trigger a rebuild
    manual_rebuild_version = "1.0.0"
  }
}
