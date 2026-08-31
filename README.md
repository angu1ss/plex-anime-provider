# plex-anime-provider

Custom metadata provider for Plex — anime from AniDB, MyAnimeList, AniList and Shikimori.
Successor to the HAMA agent for the **new Plex metadata provider API** (PMS 1.43+).

> **This is not a legacy agent.** Do not put it into the `Plug-ins` folder — it is a
> standalone HTTP service that you register in
> `Settings → Metadata Agents → Add Provider` by URL.

**Status: early development.** The provider endpoints are not implemented yet.

## Development

A local Go installation is not required — everything runs in Docker:

```sh
make help     # list targets
make test     # run tests
make up       # build and start the service (http://127.0.0.1:26463)
make release  # cross-compile binaries into dist/
```

Health check:

```sh
curl http://127.0.0.1:26463/health
```

## Endpoints

| Route      | Purpose                                                                  |
|------------|--------------------------------------------------------------------------|
| `/health`  | JSON `{status, version}` — for humans and simple monitors                |
| `/livez`   | Liveness probe: the process is alive                                     |
| `/readyz`  | Readiness probe: all conditions are met; `?verbose` lists them per check |
| `/healthz` | Legacy alias of `/livez`                                                 |

During graceful shutdown `/readyz` starts failing while `/livez` keeps
succeeding, so load balancers drain traffic before the process exits.

## Kubernetes

Example Deployment and Service with startup/liveness/readiness probes:
[`deploy/k8s-example.yaml`](deploy/k8s-example.yaml).

In containers the binary probes itself — `plex-anime-provider --healthcheck`
hits `/readyz` and exits 0/1. The Docker image and `docker-compose.yml`
already use it as `HEALTHCHECK` (distroless has no shell or curl).

## Configuration

| Flag            | Env variable                 | Default                                   |
|-----------------|------------------------------|-------------------------------------------|
| `--listen`      | `PLEX_ANIME_PROVIDER_LISTEN` | `127.0.0.1:26463`                         |
| `--healthcheck` | —                            | readiness self-check for container probes |

The default binds to loopback on purpose: the provider API has no request
authentication, so never expose the service beyond the host unless you
understand the consequences.
