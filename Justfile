set shell := ["/usr/bin/env", "bash", "-c"]
set dotenv-load := false

docker := "docker --context rosedale"
compose := docker + " compose -p rosedale -f docker-compose.yml " + `find . -type f -name 'docker-compose.yml' -not -path "*/node_modules/*" -prune -exec echo -n ' -f {}' \;`

build *SERVICES:
  {{compose}} \
    up --build -d {{SERVICES}}

deploy *SERVICES:
  {{compose}} \
    up --build -d {{SERVICES}}
