resource "docker_buildx_builder" "foo" {
  name = "foo"
  docker_container {
    image = "moby/buildkit:v0.22.0"
  }
  use = true
  bootstrap = true
}

resource "docker_image" "test_additional_contexts" {
  name = "alpine:latest"
  build {
    context      = "%s"
    dockerfile   = "%s/Dockerfile"
    force_remove = true
    builder = docker_buildx_builder.foo.name

    additional_contexts = ["second=%s"]
  }
}
