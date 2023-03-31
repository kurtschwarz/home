set shell := ["/usr/bin/env", "bash", "-c"]
set dotenv-load := true

pulumi := "pulumi logout && pulumi login && pulumi"

preview TARGET:
  pulumi --cwd {{TARGET}} preview --refresh

deploy TARGET:
  pulumi --cwd {{TARGET}} up --refresh

destroy TARGET:
  pulumi --cwd {{TARGET}} destroy
