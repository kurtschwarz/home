FROM python:3.9.5-slim@sha256:7783d80eca13fb9f8cfd8b84b27ac09ecc28f52bafdd9b943338b7e29e7741a5

ARG ANSIBLE_VERSION=2.10.7
ARG ANSIBLE_LINT_VERSION=5.0.10

ENV PATH=${PATH}:/usr/local/bin

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

ENTRYPOINT [ "ansible" ]
CMD [ "--version" ]
