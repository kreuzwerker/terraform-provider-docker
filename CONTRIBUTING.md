# Contributing

By participating in this project, you agree to abide our [code of conduct](/CODE_OF_CONDUCT.md).

## Write Issue

When you have a bug report or feature request or something, please create an issue from [here](https://github.com/kreuzwerker/terraform-provider-docker/issues/new/choose).
Before creating an issue, please check whether same or related issues exist.
Please use issue templates as much as possible.

### Guide of Bug report

* The code should be runnable for maintainers to reproduce the problem
  * We can't reproduce the problem with partial code
  * Don't include unknown input variables, local values, resources, etc
  * If you can reproduce the problem with public Docker images, please don't use private Docker images
* The code should be simple as much as possible. The simple code helps us to understand and reproduce the problem
  * Don't include unneeded resources to reproduce the problem
  * Don't set unneeded attributes to reproduce the problem

## Set up your machine

`terraform-provider-docker` is written in [Go](https://golang.org/).

Prerequisites:

- `make`, `git`, `bash`
- [Go 1.18+](https://golang.org/doc/install)
- [Docker](https://www.docker.com/)
- [Terraform 0.12+](https://terraform.io/)
- [git-chglog](https://github.com/git-chglog/git-chglog)
- [svu](https://github.com/caarlos0/svu)

Clone `terraform-provider-docker` anywhere:

```sh
git clone git@github.com:kreuzwerker/terraform-provider-docker.git
```

Install the build dependencies, tools and commit message validation:

```sh
make setup
make build
```

## Test your change

You can create a branch for your changes and try to build from the source as you go:

```sh
make build
```

### Unit and acceptance tests
When you are satisfied with the changes, **tests**, and **documentation** updates, we suggest you run:

```sh
# unit tests
make test

# acceptance test
## setup the testing environment
make testacc_setup

## run a single test
TF_LOG=INFO TF_ACC=1 go test -v ./internal/provider -timeout 60s -run ^TestAccDockerImage_data_private_config_file$

## run all test for a resource, e.g docker_container
TF_LOG=INFO TF_ACC=1 go test -v ./internal/provider -timeout 360s -run TestAccDockerContainer 

## cleanup the local testing resources
make testacc_cleanup
```

Furthermore, run the linters for the code:

```sh
# install all the dependencies
make setup
# lint the go code
make golangci-lint
```

In case you need to run the GitHub actions setup locally in a docker container and run the tests there,
run the following commands:
```sh
docker build -f testacc.Dockerfile  -t testacc-local .
docker run -it -v /var/run/docker.sock:/var/run/docker.sock -v $(pwd):/test testacc-local bash
make testacc_setup
TF_LOG=DEBUG TF_ACC=1 go test -v ./internal/provider -run ^TestAccDockerContainer_nostart$
```

### Update the documentation

Furthermore, run the generation and linters for the documentation:

```sh
# install all the dependencies
make setup
# generate or update the documentation
make website-generation
# lint the documentation
make website-link-check
make website-lint
# you can also use this command to fix most errors automatically
make website-lint-fix
```

The documentation is generated based on the tool [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs):

- The content of the `Description` attribute is parsed of each resource
- All the templates for the resources are located in `templates`.

### Test against current terraform IaC descriptions
In order to extend the provider and test it with `terraform`, build the provider as mentioned above with:

```sh
# Testing in a local mirror which needs to have the following convention.
# See https://www.terraform.io/docs/commands/cli-config.html#provider-installation for details
export TESTING_MIRROR=testing-mirror/registry.terraform.io/kreuzwerker/docker/9.9.9/$(go env GOHOSTOS)_$(go env GOHOSTARCH)
mkdir -p ./$TESTING_MIRROR

# now we build into the provider into the local mirror
go build -o ./$TESTING_MIRROR/terraform-provider-docker_v9.9.9
```

Now we change into the `testing` directory (which is ignored as well) and set an explicit version of the provider we develop:
```hcl
terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "9.9.9"
    }
  }
}

provider "docker" {
}

resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_container" "foo" {
  name    = "foo"
  image   = docker_image.foo.image_id
}
```

As the next step we can init terraform by providing a local plugin directory:
```sh
# Which reflects the convention mentioned before
# See https://www.terraform.io/docs/commands/init.html#plugin-installation
terraform init -plugin-dir=../testing-mirror
terraform plan
terraform apply -auto-approve
```

### Developing on Windows

You can build and test on Windows without `make`.  Run `go install` to
build and `Scripts\runAccTests.bat` to run the test suite.

Continuous integration for Windows is not available at the moment due
to lack of a CI provider that is free for open source projects *and*
supports running Linux containers in Docker for Windows.  For example,
AppVeyor is free for open source projects and provides Docker on its
Windows builds, but only offers Linux containers on Windows as a paid
upgrade.

## Create a commit

Commit messages should be well formatted, and to make that "standardized", we
are using Conventional Commits.

You can follow the documentation on
[their website](https://www.conventionalcommits.org).

## Submit a pull request

Push your branch to your `terraform-provider-docker` fork and open a 
pull request against the master branch.

## Releasing

Run one of the following commands (depending on the semver version you want to release): 

```sh
make patch
make minor
make major
```

Those commands will automatically:
- Replace all occurrences of the latest release, e.g. `2.11.0` with the new one, e.g. `2.12.0`: ``
- Generate the `CHANGELOG.md` 
- Regenerate the website (`make website-generation`)
