# syntax=docker/dockerfile:1.2.1-labs

# --- global arguments ----------------------------------------------------------------------------

ARG OP_CLI_VERSION="2.32.0@sha256:57d7d6a2bb2b74b2cf8111f6afb2973c74772198f82ea30359a53faae9fff5b1"
ARG ANSIBLE_VERSION="2.18.6@sha256:edba669e24f558c5c0e22c73d3a2c71dd77781176d84c0a247f44ebd66f6f643"

# --- layers --------------------------------------------------------------------------------------

FROM 1password/op:${OP_CLI_VERSION} AS op-cli

FROM alpine/ansible:${ANSIBLE_VERSION} AS ansible

COPY --from=op-cli /usr/local/bin/op /usr/local/bin/op
