# AGENTS.md

Guidance for AI agents operating in this repository. `CLAUDE.md` is a symlink to
this file.

## Overview

This is a **megarepo for a personal [homelab](https://www.reddit.com/r/homelab/wiki/introduction/)**.
It contains the infrastructure-as-code, configuration management, and
documentation used to provision and operate a small Kubernetes (k3s) cluster
running on Raspberry Pi 5, ZimaBoard 2, and RackNerd KVM hosts.

The three pillars of the repo:

- **`ansible/`** — provisions the bare hosts (WireGuard, k3s cluster).
- **`pulumi/`** — deploys cluster workloads (Helm charts) via Pulumi.
- **`docs/`** — a Next.js (Nextra) documentation site.

## Tooling & Conventions

- **Task runner:** [`just`](https://just.systems/). The root `justfile` imports
  per-area modules (`ansible`, `pulumi`). Prefer `just` recipes over invoking
  tools directly.
- **Everything runs in Docker.** Ansible and Pulumi execute inside containers
  via `docker compose` (see `compose.yaml`, which includes each area's
  `compose.partial.yaml`). Do not assume `ansible`, `pulumi`, or the Go/Node
  toolchains are installed on the host.
- **Secrets:** managed with 1Password CLI (`op`) and Ansible Vault. `.env`,
  `.filen-cli-auth-config`, and `Pulumi.<stack>.yaml` are secret-bearing and
  gitignored / mode `600`. Never commit or print secrets.
- **Node.js:** version pinned in `.nvmrc` (25.2.1); package manager is
  `pnpm` (workspace defined in `pnpm-workspace.yaml`), with `turbo` for the
  docs build.
- **Languages:** Go (Pulumi projects), YAML (Ansible), TypeScript/JS + MDX
  (docs).

## Directory Layout

```
ansible/            # host provisioning
  inventories/      # cluster/ and external/ inventories (inventory.yaml)
  roles/            # k3s, pi5, wireguard, zb2
  provisioning.yaml # main playbook; also upgrade.yaml, teardown.yaml
  justfile          # `just ansible playbook ...`, inventory, vault, shell
pulumi/             # cluster workload deployments
  projects/         # one dir per Pulumi project (Go), e.g. kube-systems-cilium
  resources/        # shared resources
  justfile          # `just pulumi up <project>`, preview, stack, login, ...
docs/               # Next.js/Nextra documentation site (content/*.mdx)
compose.yaml        # top-level docker compose, includes the partials
justfile            # root recipes: docs, build; imports ansible & pulumi
```

## Common Commands

Run from the repo root unless noted.

- `just docs` — start the docs site at http://localhost:3000/docs/
- `just build <target>` — build a compose service image
- `just ansible playbook provisioning.yaml` — run the provisioning playbook
- `just ansible inventory` — dump the resolved inventory
- `just ansible vault ...` — Ansible Vault operations
- `just pulumi preview <project>` / `just pulumi up <project>` — plan/apply a
  Pulumi project (e.g. `kube-systems-cilium`)
- `just pulumi shell` — open a shell in the Pulumi container

## Cluster Facts

- **k3s** cluster; server node(s) are Pi 5, agents are ZimaBoard 2 and a
  RackNerd KVM host. See `ansible/inventories/cluster/inventory.yaml`.
- Default k3s add-ons **disabled**: traefik, metrics-server, servicelb,
  kube-proxy; `flannel-backend: none` — **Cilium** provides CNI + kube-proxy
  replacement (see `pulumi/projects/kube-systems-cilium`).
- Network CIDRs: pods `10.33.0.0/16`, services `10.34.0.0/16`, external
  `10.35.0.0/16`; cluster VLAN `10.32.0.0/14`.

## Working Notes for Agents

- When adding a cluster workload, create a new `pulumi/projects/<name>/` Go
  project mirroring the existing ones (Helm chart deployed against the
  `homelab` kube context/provider) and add a stack config `Pulumi.homelab.yaml`.
- When changing host configuration, edit the relevant Ansible role and
  inventory rather than SSHing manually.
- Prefer editing existing files and matching the surrounding style. Keep
  secrets out of version control.
