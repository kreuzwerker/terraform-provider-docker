resource "docker_image" "zoo" {
  name = "zoo"
  build {
    path = "."
    tag  = ["zoo:develop"]
    build_arg = {
      foo : "zoo"
    }
    label = {
      author : "zoo"
    }
  }
}