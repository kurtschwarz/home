set shell := ["/usr/bin/env", "bash", "-c"]
set dotenv-load := true

pulumi := "pulumi logout && pulumi login && pulumi"

refresh TARGET:
  {{pulumi}} --cwd {{TARGET}} refresh --stack dev

preview TARGET:
  {{pulumi}} --cwd {{TARGET}} preview --stack dev --refresh --diff

deploy TARGET:
  {{pulumi}} --cwd {{TARGET}} up --stack dev --refresh

destroy TARGET *ARGS:
  {{pulumi}} --cwd {{TARGET}} destroy --stack dev {{ARGS}}

pulumi TARGET *ARGS:
  {{pulumi}} --cwd {{TARGET}} {{ARGS}}
