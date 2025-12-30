# syntax=docker/dockerfile:1.4

# --- global arguments ----------------------------------------------------------------------------

ARG UID=1100
ARG GID=1100

ARG OP_CLI_VERSION="2.32.0@sha256:57d7d6a2bb2b74b2cf8111f6afb2973c74772198f82ea30359a53faae9fff5b1"
ARG GO_VERSION="1.25.5-trixie@sha256:8e8f9c84609b6005af0a4a8227cee53d6226aab1c6dcb22daf5aeeb8b05480e1"
ARG PULUMI_VERSION="3.213.0@sha256:c7abcdd9e5249cd4a3bb6abef51c84b041d004ff130d593836acf37edba23ead"
ARG KUBECTL_VERSION="v1.35.0@sha256:63a9f14d017459f47d23cb266b2857e82ffe6602d8955eef00ce111b2d8a61cf"

# --- layers --------------------------------------------------------------------------------------

FROM 1password/op:${OP_CLI_VERSION} AS op-cli

FROM rancher/kubectl:${KUBECTL_VERSION} AS kubectl

FROM golang:${GO_VERSION} AS go

FROM pulumi/pulumi-base:${PULUMI_VERSION} AS pulumi

COPY --from=op-cli /usr/local/bin/op /usr/local/bin/op
COPY --from=kubectl /bin/kubectl /usr/local/bin/kubectl
COPY --from=go /usr/local/go /usr/local/go

ENV PATH=${PATH}:/usr/local/go/bin

COPY ./ /workspace/
WORKDIR /workspace
