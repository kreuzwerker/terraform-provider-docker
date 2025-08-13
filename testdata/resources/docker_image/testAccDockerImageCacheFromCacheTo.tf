resource "docker_buildx_builder" "foo" {
  name = "foo"
  docker_container {
    image = "moby/buildkit:v0.22.0"
  }
  use = true
  bootstrap = true
}

resource "docker_image" "test_cache_to" {
  name = "alpine:latest"
  build {
    context      = "%s"
    dockerfile   = "%s/Dockerfile"
    force_remove = true
    builder = docker_buildx_builder.foo.name

    cache_to = ["type=local,mode=min,dest=/tmp/cache"]
  }
}

resource "docker_image" "test_cache_from" {
  name = "alpine:latest2"
  build {
    context      = "%s"
    dockerfile   = "%s/Dockerfile"
    force_remove = true
    builder = docker_buildx_builder.foo.name

    cache_from = ["type=local,src=/tmp/cache"]
  }
  depends_on = [ docker_image.test_cache_to ]
}
