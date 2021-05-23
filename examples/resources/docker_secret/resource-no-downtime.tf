resource "docker_secret" "service_secret" {
  name = "${var.service_name}-secret-${replace(timestamp(), ":", ".")}"
  data = base64encode(
    templatefile("${path.cwd}/foo.secret.json.tpl",
      {
        secret = "s3cr3t"
      }
    )
  )

  lifecycle {
    ignore_changes        = ["name"]
    create_before_destroy = true
  }
}

resource "docker_service" "service" {
  # ...
  secrets = [
    {
      secret_id   = docker_secret.service_secret.id
      secret_name = docker_secret.service_secret.name
      file_name   = "/root/configs/configs.json"
    },
  ]
}
