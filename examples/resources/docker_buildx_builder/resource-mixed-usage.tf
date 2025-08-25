# Example showing different auto_recreate settings for different environments
resource "docker_buildx_builder" "development" {
  name          = "dev-builder"
  driver        = "docker-container"
  bootstrap     = true
  auto_recreate = true # Enable for development - convenience

  docker_container {
    image = "moby/buildkit:latest"
  }

  platform = ["linux/amd64"]
}

resource "docker_buildx_builder" "production" {
  name          = "prod-builder"
  driver        = "docker-container"
  bootstrap     = true
  auto_recreate = false # Disable for production - explicit control

  docker_container {
    image        = "moby/buildkit:v0.22.0" # Pinned version for production
    default_load = false
  }

  platform = ["linux/amd64", "linux/arm64"]
}

resource "docker_buildx_builder" "k8s_builder" {
  name          = "k8s-builder"
  driver        = "kubernetes"
  bootstrap     = true
  auto_recreate = true # Enable for k8s environments

  kubernetes {
    namespace = "buildkit"
    image     = "moby/buildkit:latest"
    replicas  = 2

    requests {
      cpu    = "100m"
      memory = "256Mi"
    }

    limits {
      cpu    = "500m"
      memory = "512Mi"
    }
  }

  platform = ["linux/amd64", "linux/arm64"]
}
