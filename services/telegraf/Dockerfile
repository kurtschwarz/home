FROM docker.io/library/telegraf:1.25.3@sha256:c0ea630ccdce911024440adc04b252ed31c305dcc39e79e73d4da7c5e7d5950a AS telegraf

RUN apt-get update && \
    apt-get install -y \
      nvme-cli \
      lm-sensors \
      ipmitool \
      smartmontools

ADD ./telegraf.conf /etc/telegraf/telegraf.conf
