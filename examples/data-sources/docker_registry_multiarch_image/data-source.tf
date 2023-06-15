### Must be a Docker multi-arch image
data "docker_registry_multiarch_image" "alpine" {
  name = "alpine:latest"
}
