resource "docker_volume" "test_volume" {
  name = "tftest-volume"
}

resource "docker_volume" "test_volume_2" {
  name = "tftest-volume2"
}

resource "docker_config" "service_config" {
  name = "tftest-full-myconfig"
  data = "ewogICJwcmVmaXgiOiAiMTIzIgp9"
}

resource "docker_secret" "service_secret" {
  name = "tftest-mysecret"
  data = "ewogICJrZXkiOiAiUVdFUlRZIgp9"
}

resource "docker_network" "test_network" {
  name   = "tftest-network"
  driver = "overlay"
}

resource "docker_service" "foo" {
  name = "tftest-service-basic"

  task_spec {
    container_spec {
      image = "repo.mycompany.com:8080/foo-service:v1"

      labels {
        label = "foo.bar"
        value = "baz"
      }

      command  = ["ls"]
      args     = ["-las"]
      hostname = "my-fancy-service"

      env = {
        MYFOO = "BAR"
      }

      dir    = "/root"
      user   = "root"
      groups = ["docker", "foogroup"]

      privileges {
        se_linux_context {
          disable = true
          user    = "user-label"
          role    = "role-label"
          type    = "type-label"
          level   = "level-label"
        }
      }

      read_only = true

      mounts {
        target    = "/mount/test"
        source    = docker_volume.test_volume.name
        type      = "bind"
        read_only = true

        bind_options {
          propagation = "rprivate"
        }
      }

      mounts {
        target    = "/mount/test2"
        source    = docker_volume.test_volume_2.name
        type      = "volume"
        read_only = true

        volume_options {
          no_copy = true
          labels {
            label = "foo"
            value = "bar"
          }
          driver_name = "random-driver"
          driver_options = {
            op1 = "val1"
          }
        }
      }

      stop_signal       = "SIGTERM"
      stop_grace_period = "10s"

      healthcheck {
        test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
        interval = "5s"
        timeout  = "2s"
        retries  = 4
      }

      hosts {
        host = "testhost"
        ip   = "10.0.1.0"
      }

      dns_config {
        nameservers = ["8.8.8.8"]
        search      = ["example.org"]
        options     = ["timeout:3"]
      }

      secrets {
        secret_id   = docker_secret.service_secret.id
        secret_name = docker_secret.service_secret.name
        file_name   = "/secrets.json"
        file_uid    = "0"
        file_gid    = "0"
        file_mode   = 0777
      }

      secrets {
        # another secret
      }

      configs {
        config_id   = docker_config.service_config.id
        config_name = docker_config.service_config.name
        file_name   = "/configs.json"
      }

      configs {
        # another config
      }
    }

    resources {
      limits {
        nano_cpus    = 1000000
        memory_bytes = 536870912
      }

      reservation {
        nano_cpus    = 1000000
        memory_bytes = 536870912

        generic_resources {
          named_resources_spec = [
            "GPU=UUID1",
          ]

          discrete_resources_spec = [
            "SSD=3",
          ]
        }
      }
    }

    restart_policy = {
      condition    = "on-failure"
      delay        = "3s"
      max_attempts = 4
      window       = "10s"
    }

    placement {
      constraints = [
        "node.role==manager",
      ]

      prefs = [
        "spread=node.role.manager",
      ]

      max_replicas = 1
    }

    force_update = 0
    runtime      = "container"
    networks     = [docker_network.test_network.id]

    log_driver {
      name = "json-file"

      options = {
        max-size = "10m"
        max-file = "3"
      }
    }
  }

  mode {
    replicated {
      replicas = 2
    }
  }

  update_config {
    parallelism       = 2
    delay             = "10s"
    failure_action    = "pause"
    monitor           = "5s"
    max_failure_ratio = "0.1"
    order             = "start-first"
  }

  rollback_config {
    parallelism       = 2
    delay             = "5ms"
    failure_action    = "pause"
    monitor           = "10h"
    max_failure_ratio = "0.9"
    order             = "stop-first"
  }

  endpoint_spec {
    mode = "vip"

    ports {
      name           = "random"
      protocol       = "tcp"
      target_port    = "8080"
      published_port = "8080"
      publish_mode   = "ingress"
    }

    ports {
      # another port
    }
  }
}
