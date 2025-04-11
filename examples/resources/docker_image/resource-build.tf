resource "docker_image" "zoo" {
  name = "zoo"
  build {
    context = "."
    tag     = ["zoo:develop"]
    build_args = {
      foo : "zoo"
    }
    label = {
      author : "zoo"
    }
  }
}