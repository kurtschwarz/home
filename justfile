mod ansible './ansible/justfile'
mod pulumi './pulumi/justfile'

set shell := ["/usr/bin/env", "bash", "-sec"]
set dotenv-load := true

docker := `which docker`
compose := docker + ' compose'

docs *ARGS:
  {{ compose }} up docusaurus {{ARGS}}
