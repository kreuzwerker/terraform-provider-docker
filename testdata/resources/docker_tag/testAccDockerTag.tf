resource "docker_image" "foo" {
  name = "nginx:1.17.6@sha256:36b77d8bb27ffca25c7f6f53cadd059aca2747d46fb6ef34064e31727325784e"
}

resource "docker_tag" "foobar" {
    source_image = "%s"
    target_image = "%s"
    depends_on = [
      docker_image.foo
    ]
}
