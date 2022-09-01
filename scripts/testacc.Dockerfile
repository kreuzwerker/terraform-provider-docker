FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive
ARG DOCKER_CE_VERSION="5:20.10.10~3-0~ubuntu-focal"
ARG GOLANG_VERSION="1.18"
ARG TERRAFORM_VERSION="0.15.2"

# Install the baseline
RUN apt-get update && \
    apt-get -y install apt-transport-https ca-certificates curl gnupg-agent software-properties-common build-essential

# Install golang
RUN curl -L https://dl.google.com/go/go${GOLANG_VERSION}.linux-amd64.tar.gz > go${GOLANG_VERSION}.linux-amd64.tar.gz && \
    tar xzf go${GOLANG_VERSION}.linux-amd64.tar.gz && \
    rm -f go${GOLANG_VERSION}.linux-amd64.tar.gz
ENV GOPATH /go
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

# Install docker
# see listed docker versions: 'apt-cache policy docker-ce'
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" && \
    apt-get update && \
    apt-cache policy docker-ce
RUN apt-get -y install docker-ce=${DOCKER_CE_VERSION}

# Install terraform
RUN curl -fsSL https://apt.releases.hashicorp.com/gpg | apt-key add - && \
    apt-add-repository "deb [arch=$(dpkg --print-architecture)] https://apt.releases.hashicorp.com $(lsb_release -cs) main" && \
    apt-get update
RUN apt-get -y install terraform=${TERRAFORM_VERSION}