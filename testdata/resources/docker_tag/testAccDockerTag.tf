resource "docker_tag" "foobar" {
    source_image = "%s"
    target_image = "%s"
}