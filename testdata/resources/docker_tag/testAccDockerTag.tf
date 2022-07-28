resource "docker_image" "foo" {
  name = "nginx:1.17.6"
}

resource "docker_tag" "foobar" {
    source_image = "%s"
    target_image = "%s"
    depends_on = [
      docker_image.foo
    ]
}
