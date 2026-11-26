provider "docker" {}

resource "docker_image" "nginx" {
  name = "nginx:1.29"

  daemon {
    host     = "ssh://user@${hcloud_server.web.ipv4_address}:22"
    ssh_opts = ["-o", "StrictHostKeyChecking=accept-new"]
  }
}

resource "docker_container" "nginx" {
  name  = "nginx"
  image = docker_image.nginx.image_id

  daemon {
    host     = "ssh://user@${hcloud_server.web.ipv4_address}:22"
    ssh_opts = ["-o", "StrictHostKeyChecking=accept-new"]
  }
}
