set shell := ["/usr/bin/env", "bash", "-c"]
set dotenv-load := false

deploy TARGET:
  pulumi --cwd {{TARGET}} up --refresh
