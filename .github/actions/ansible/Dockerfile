FROM golang:1.16.4-buster@sha256:ab32429d40c1b734ed4f036838ac516352182f9414563478fa88a1553c4a4414 AS builder

ENV GO111MODULE=off \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build/
COPY ./ /build/
RUN go build -o ansible-action

FROM python:3.9.5-slim@sha256:7783d80eca13fb9f8cfd8b84b27ac09ecc28f52bafdd9b943338b7e29e7741a5 AS ansible

ARG ANSIBLE_VERSION=2.10.7
ARG ANSIBLE_LINT_VERSION=5.0.10

ENV PATH=${PATH}:/usr/local/bin

COPY --from=builder /build/ansible-action /usr/local/bin

RUN set -ex ; \
    \
    apt-get update -y && \
    apt-get install -y \
      ca-certificates \
      openssh-client && \
    \
    pip install \
      ansible==${ANSIBLE_VERSION} \
      ansible-lint==${ANSIBLE_LINT_VERSION} && \
    \
    mkdir -p /etc/ansible && \
    echo "localhost" > /etc/ansible/hosts && \
    \
    echo -e "\n\n" >> /etc/ssh/ssh_config && \
    echo -e "Host *\n" >> /etc/ssh/ssh_config && \
    echo -e "    StrictHostKeyChecking no\n" >> /etc/ssh/ssh_config && \
    echo -e "    UserKnownHostsFile=/dev/null\n" >> /etc/ssh/ssh_config

ENTRYPOINT [ "/usr/local/bin/ansible-action" ]
CMD []
