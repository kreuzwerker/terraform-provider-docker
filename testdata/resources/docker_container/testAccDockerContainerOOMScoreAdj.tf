resource "docker_image" "oom" {
  name = "nginx:latest"
}

resource "docker_container" "oom_application" {
  name          = "%[2]s-application"
  image         = docker_image.oom.image_id
  oom_score_adj = %[1]d
}

resource "docker_container" "oom_database" {
  name          = "%[2]s-database"
  image         = docker_image.oom.image_id
  oom_score_adj = -500
}

resource "docker_container" "oom_cache" {
  name          = "%[2]s-cache"
  image         = docker_image.oom.image_id
  oom_score_adj = -250
}

resource "docker_container" "oom_ml" {
  name          = "%[2]s-ml"
  image         = docker_image.oom.image_id
  oom_score_adj = 500
}

resource "docker_container" "oom_zero" {
  name          = "%[2]s-zero"
  image         = docker_image.oom.image_id
  oom_score_adj = 0
}

resource "docker_container" "oom_unset" {
  name  = "%[2]s-unset"
  image = docker_image.oom.image_id
}
