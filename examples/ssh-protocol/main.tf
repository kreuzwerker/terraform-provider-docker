# test case
provider "docker" {
  version = "~> 1.2.0"
  alias   = "test"

  host = "ssh://root@localhost:32822"
}

resource "docker_image" "test" {
  provider = "docker.test"
  name     = "busybox:latest"
}

# scaffolding
variable "pub_key" {
  type = string
}

provider "docker" {
  version = "~> 1.2.0"
}

resource "docker_image" "dind" {
  name = "docker:18.09.0-dind"
}

resource "docker_container" "dind" {
  depends_on = [
    "docker_image.dind",
  ]

  name  = "dind"
  image = "docker:18.09.0-dind"

  privileged = true

  start = true

  command = ["/bin/sh", "-c",
    <<SH
    set -e
    apk --no-cache add openrc
    
    # setup sshd
    apk --no-cache add openssh-server
    rc-update add sshd

    # setup dockerd
    apk --no-cache add docker-openrc
    echo DOCKERD_BINARY=/usr/local/bin/dockerd > /etc/conf.d/docker
    echo DOCKERD_OPTS=--host=unix:///var/run/docker.sock >> /etc/conf.d/docker
    rc-update add docker

    # setup ssh for root
    mkdir -p ~/.ssh

    # link docker cli so root can see it
    ln -s /usr/local/bin/docker /usr/bin/

    # start ssh and docker
    exec /sbin/init
    SH
    ,
  ]

  ports {
    internal = 22
    external = 32822
  }

  upload {
    content = <<AUTHORIZED_KEYS
      ${var.pub_key}
      AUTHORIZED_KEYS

    file = "/root/.ssh/authorized_keys"
  }
}
