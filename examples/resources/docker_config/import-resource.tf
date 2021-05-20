resource "docker_config" "foo" {
  name = "foo"
  data = base64encode("{\"a\": \"b\"}")
}
