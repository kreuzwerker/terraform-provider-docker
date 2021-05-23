resource "docker_config" "foo_config" {
  name = "foo_config"
  data = base64encode(
    templatefile("${path.cwd}/foo.config.json.tpl",
      {
        port = 8080
      }
    )
  )
}