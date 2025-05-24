resource "docker_image" "test" {
  name = "ubuntu:11"
  build {
    context      = "."
    dockerfile   = "Dockerfile"
    force_remove = true
    # both commented values are the default values, leaving them there for visibility
    # builder = "default"
    # platform = "linux/amd64"

    build_log_file = "%s"
  }
}
