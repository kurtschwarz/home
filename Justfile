set shell := ["/usr/bin/env", "bash"]

build *TARGETS:
  #!/usr/bin/env bash

  if [[ ! -z "{{TARGETS}}" ]] ; then
    for target in {{TARGETS}} ; do
      if [[ -f "./services/${target}/justfile" ]] ; then
        if [[ $(just -f ./services/${target}/justfile --list | grep -e "^ *build") ]] ; then
          just -f ./services/${target}/justfile build
          continue
        fi
      fi
    done
  fi
