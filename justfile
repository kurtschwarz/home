mod metal './metal/justfile'

set shell := ["/usr/bin/env", "bash", "-sec"]
set dotenv-load := true

docker := `which docker`
compose := docker + ' compose'

pulumi := `which pulumi`

[private]
init-pulumi:
  #!/usr/bin/env bash
  set -exuo pipefail

  pulumi logout
  pulumi login

  if [[ ! $(pulumi whoami | grep "kurtschwarz") ]] ; then
    exit 1
  fi

refresh TARGET: (init-pulumi)
  {{pulumi}} --cwd {{TARGET}} refresh --stack dev

preview TARGET: (init-pulumi)
  {{pulumi}} --cwd {{TARGET}} preview --stack dev --refresh --diff

deploy TARGET: (init-pulumi)
  {{pulumi}} --cwd {{TARGET}} up --stack dev --refresh

destroy TARGET *ARGS: (init-pulumi)
  {{pulumi}} --cwd {{TARGET}} destroy --stack dev {{ARGS}}

pulumi TARGET *ARGS: (init-pulumi)
  {{pulumi}} --cwd {{TARGET}} {{ARGS}}

docs *ARGS:
  {{ compose }} up docusaurus {{ARGS}}
