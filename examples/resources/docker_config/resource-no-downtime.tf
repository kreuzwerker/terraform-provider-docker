resource "docker_config" "service_config" {
  name = "${var.service_name}-config-${replace(timestamp(), ":", ".")}"
  data = base64encode(
    templatefile("${path.cwd}/foo.config.json.tpl",
      {
        port = 8080
      }
    )
  )

  lifecycle {
    ignore_changes        = ["name"]
    create_before_destroy = true
  }
}

resource "docker_service" "service" {
  # ... other attributes omitted for brevity
  configs {
    config_id   = docker_config.service_config.id
    config_name = docker_config.service_config.name
    file_name   = "/root/configs/configs.json"
  }
}
