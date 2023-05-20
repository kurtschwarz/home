set shell := ["/usr/bin/env", "bash", "-c"]
set dotenv-load := true

pulumi := "pulumi logout && pulumi login && pulumi"

refresh TARGET:
  pulumi --cwd {{TARGET}} refresh --stack dev

preview TARGET:
  pulumi --cwd {{TARGET}} preview --stack dev --refresh

deploy TARGET:
  pulumi --cwd {{TARGET}} up --stack dev --refresh

destroy TARGET:
  pulumi --cwd {{TARGET}} destroy --stack dev
