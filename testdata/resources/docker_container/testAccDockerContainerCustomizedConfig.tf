resource "docker_image" "foo" {
  name = "nginx:latest"
}

resource "docker_container" "foo" {
  name                  = "tf-test"
  image                 = docker_image.foo.image_id
  entrypoint            = ["/bin/bash", "-c", "cat /proc/kmsg"]
  user                  = "root:root"
  restart               = "on-failure"
  destroy_grace_seconds = 10
  max_retry_count       = 5
  memory                = 512
  shm_size              = 128
  memory_swap           = 2048
  cpu_shares            = 32
  cpu_set               = "0-1"

  capabilities {
    add  = ["ALL"]
    drop = ["SYS_ADMIN"]
  }

  security_opts = ["apparmor=unconfined", "label=disable"]

  dns        = ["8.8.8.8"]
  dns_opts   = ["rotate"]
  dns_search = ["example.com"]
  labels {
    label = "env"
    value = "prod"
  }
  labels {
    label = "role"
    value = "test"
  }
  labels {
    label = "maintainer"
    value = "NGINX Docker Maintainers <docker-maint@nginx.com>"
  }
  log_driver = "json-file"
  log_opts = {
    max-size = "10m"
    max-file = 20
  }
  network_mode = "bridge"

  # Disabled for tests due to
  # --storage-opt is supported only for overlay over xfs with 'pquota' mount option
  # see https://github.com/kreuzwerker/terraform-provider-docker/issues/177
  # storage_opts = {
  #   size = "120Gi"
  # }

  networks_advanced {
    name    = docker_network.test_network.name
    aliases = ["tftest"]
  }

  host {
    host = "testhost"
    ip   = "10.0.1.0"
  }

  host {
    host = "testhost2"
    ip   = "10.0.2.0"
  }

  ulimit {
    name = "nproc"
    hard = 1024
    soft = 512
  }

  ulimit {
    name = "nofile"
    hard = 262144
    soft = 200000
  }

  pid_mode    = "host"
  userns_mode = "testuser:231072:65536"
  ipc_mode    = "private"
  working_dir = "/tmp"
}

resource "docker_network" "test_network" {
  name = "test"
}
