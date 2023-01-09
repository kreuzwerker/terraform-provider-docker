resource "docker_image" "test" {
  name = "ubuntu:11"
  build {
    context      = "."
    dockerfile   = "Dockerfile"
    force_remove = true
    build_arg = {
      test_arg = "kenobi"
    }
    label = {
      test_label1 = "han"
      test_label2 = "solo"
    }
  }
}
