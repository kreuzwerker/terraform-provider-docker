resource "docker_config" "foo_config" {
  name = "foo_config"
  data = base64encode("{\"a\": \"b\"}")
}