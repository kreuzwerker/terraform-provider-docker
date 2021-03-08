module github.com/terraform-providers/terraform-provider-docker

require (
	github.com/docker/cli v20.10.0-beta1.0.20201029214301-1d20b15adc38+incompatible // v19.03.8
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.0+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/hashicorp/terraform-plugin-sdk v1.16.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.4.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/buildkit v0.8.1 // indirect
	github.com/moby/sys/symlink v0.1.0 // indirect
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/opencontainers/image-spec v1.0.1
)

go 1.15
