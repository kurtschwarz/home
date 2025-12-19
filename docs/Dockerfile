# syntax=docker/dockerfile:1.2.1-labs

# --- global arguments ----------------------------------------------------------------------------

ARG NODE_BUILD_VERSION="25.2.1-trixie@sha256:7942b337b20b67cd846828242f4e033cc29a83dde0f179b373636f3cbf072ae5"

# --- base layer ----------------------------------------------------------------------------------

FROM node:${NODE_BUILD_VERSION} AS base

ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"

RUN apt-get update --yes && \
    apt-get install --yes --no-install-recommends \
      "jq=1.*" && \
    \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
COPY ./package.json /workspace
RUN npm install --global $(jq -r .packageManager package.json) && \
    pnpm install --global turbo@$(jq -r .devDependencies.turbo package.json)

# -------------------------------------------------------------------------------------------------

FROM base AS turbo-prune

COPY . /workspace
RUN turbo prune --docker --scope="@kurtschwarz/homelab-docs"

# -------------------------------------------------------------------------------------------------

FROM base AS dependencies

COPY --from=turbo-prune /workspace/out/json/ /workspace
COPY --from=turbo-prune /workspace/out/pnpm-lock.yaml /workspace/pnpm-lock.yaml
COPY ./tsconfig*.json /workspace/

RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --prefer-offline --frozen-lockfile

# -------------------------------------------------------------------------------------------------

FROM base as dev
COPY --from=dependencies /workspace /workspace
COPY --from=turbo-prune /workspace/out/full /workspace
COPY ./docs /workspace/docs/

WORKDIR /workspace/docs/
ENTRYPOINT [ "pnpm" ]
CMD [ "run", "dev" ]
