## The following code runs `docker system prune` when the trigger resource changes.

resource "terraform_data" "prune_trigger" {
  triggers_replace = [
    timestamp()
  ]

  lifecycle {
    action_trigger {
      events  = [after_update]
      actions = [action.docker_system_prune.cleanup]
    }
  }
}

action "docker_system_prune" "cleanup" {
  config {
    all     = true
    volumes = true
    filter = [
      "label!=terraform-manage",
    ]
  }
}
