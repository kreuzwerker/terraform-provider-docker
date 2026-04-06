data "docker_containers" "this" {}

output "container_fingerprint" {
  description = "Toy example mapping Container IDs to an md5sum fingerprint of its configuration"
  value       = { for c in data.docker_containers.this.containers : c.id => md5(jsonencode(c)) }
}

