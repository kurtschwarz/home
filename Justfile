set shell := ["/usr/bin/env", "bash", "-c"]
set dotenv-load := false

docker := "docker -H ssh://root@192.168.86.8"
compose := docker + " compose -p rosedale -f docker-compose.yml " + `find . -type f -name 'docker-compose.yml' -not -path "*/node_modules/*" -prune -exec echo -n ' -f {}' \;`

deploy *SERVICES:
  {{compose}} \
    up --build -d {{SERVICES}}

logs *SERVICES:
  {{compose}} \
    logs -ft {{SERVICES}}
