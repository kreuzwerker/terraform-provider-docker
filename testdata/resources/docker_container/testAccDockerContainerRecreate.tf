data "docker_registry_image" "redis" {
	name = "redis:7-alpine"
}

resource "docker_image" "redis" {
	name = "redis:7-alpine"
	keep_locally = true
	pull_triggers = [data.docker_registry_image.redis.sha256_digest]
}

resource "docker_container" "redis-container-2" {
	image = docker_image.redis.image_id
	name = "redis-2"
	hostname = "redis-2"


	restart = "unless-stopped"
}