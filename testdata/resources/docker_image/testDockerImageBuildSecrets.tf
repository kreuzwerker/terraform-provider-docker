resource "docker_image" "test" {
  name = "ubuntu:11"
  build {
    context      = "."
    dockerfile   = "Dockerfile"
    force_remove = true
    builder = "default"
    platform = "linux/amd64"

    secrets {
      id  = "TEST_SECRET_SRC"
      src = "./secret"
    }

    secrets {
      id  = "TEST_SECRET_ENV"
      env = "PATH"
    }
  }
}
