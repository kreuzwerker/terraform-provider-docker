FROM debian:9.13-slim

RUN groupadd -r testgroup && useradd -r testuser -G testgroup


COPY --chown=testuser:testgroup . /testroot

USER testuser

WORKDIR /testroot
RUN cat testfile
