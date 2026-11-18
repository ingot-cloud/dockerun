# Dockerun

**English** · [简体中文](README.zh-CN.md)

Dockerun is a lightweight, self-hosted console for managing Docker resources and releasing applications. It brings a private image registry, application deployments, and offline delivery together for a single Docker host or a fixed group of deployment targets.

The Dockerun image includes a Chinese-language web console, a Go API, an embedded CNCF Distribution registry, Docker CLI, and Docker Compose. SQLite is the default metadata store. An existing MySQL instance can also be used.

[Features](#features) · [Deploy from source](#deploy-from-source) · [First application deployment](#first-application-deployment) · [Local development](#local-development) · [Documentation](#documentation) · [License](#license)

## Features

| Area | What you can do |
| --- | --- |
| Containers | Start, stop, restart, force-stop, pause, resume, and remove containers. Stream logs, inspect configuration, view process lists and resource usage, and open exec or attach consoles. Rename containers, update restart policies, and commit containers to images. |
| Local images | Pull, tag, import, export, and delete images. Inspect image metadata, build history, and which containers use an image. |
| Networks and volumes | Create, inspect, and remove Docker networks and volumes. Connect or disconnect containers and see resource usage relationships. |
| Built-in and external registries | Use standard `docker login`, `push`, and `pull` with the embedded private registry. Browse repositories, tags, and digests, and manage image manifests subject to retention protections. Connect to existing registries, test connections, and save encrypted credentials. |
| Applications and environments | Define deployments with Docker Compose or Bash commands, including configuration files and pre- and post-deployment scripts. Set environment parameters, script overrides, container associations, and checks. Preview changes before submitting a deployment. |
| Application templates | Start with templates for a single service, MySQL, Redis, Nacos, Kafka, Registry, Portainer, Nginx Proxy, and Ingot services. Save custom templates and preview, import, or export standard ZIP packages. |
| Releases and deployment history | Keep versioned deployment definitions and track queued tasks, per-host results, logs, and health checks. Redeploy or promote a saved release and check the actual running image and Image ID. Automatic recovery is available for eligible stateless Compose deployments when enabled. |
| Automation and remote targets | Trigger releases through authorized image pushes, configure external registry webhooks, and use scoped tokens with the CLI for CI/CD. Run deployments on the local host or through outbound-connected Agents. Fixed host groups deploy sequentially and stop on failure. |
| Offline delivery | Import `docker save` archives or complete release bundles containing images, Compose, and non-secret configuration. Upload in resumable chunks or import from a configured server directory. The CLI can package releases for the target platform. |
| Administration and maintenance | Manage users, registry permissions, automation tokens, and Agent credentials. View system and Docker status and audit events. Run legacy scripts manually, download full backups, restore from the CLI, and run registry garbage collection during read-only maintenance. |

Docker resource pages operate on the connected Docker Engine. Agents execute deployments on remote targets. Fixed host groups run one target at a time. Replacing containers can cause downtime, and recovery applies to the deployment definition and images. Back up application data volumes separately. [Deployment workflow](docs/INTEGRATED_DEPLOYMENT.md) · [Verification and compatibility](docs/VALIDATION.md)

## Deploy from source

### Requirements

- Linux Docker Engine with API 1.44 or later (Docker 25+) and Docker Compose v2.
- A Linux `amd64` or `arm64` target. Use a builder that supports the target platform for cross-platform builds.
- Docker socket access and internet access on the build machine to download base images and dependencies. The image build installs the Go, Node.js, and pnpm toolchains inside its build stages.

### Build and start

Run these commands from a terminal on the build and deployment host.

```bash
git clone https://github.com/ingot-cloud/dockerun.git
cd dockerun
cp .env.example .env
```

Edit `.env`. For an HTTP test on the host itself, use these values.

```dotenv
PUBLIC_URL=http://localhost:8080
DOCKERUN_IMAGE=dockerun:2.0.0
DATABASE_TYPE=sqlite
```

For access from another machine, use the server's actual DNS name or IP address in `PUBLIC_URL`. Use a root URL reachable by both clients and Docker daemons, including the published port when required. Docker uses this address for registry authentication. HTTPS configuration is covered below.

```bash
docker compose config --quiet
docker compose build
docker compose up -d --wait --no-build --pull never
docker compose ps
docker compose logs --tail=100 dockerun
curl --fail http://127.0.0.1:8080/health
```

Open the configured URL in a browser, set the password for `admin` on first use, and sign in. Confirm Docker connectivity on the System Information page. The example uses host port `8080`. If you change it, update the Compose port mapping and the corresponding public or proxy URL.

### Production, storage, and offline installation

- **HTTPS.** Use a trusted certificate and an HTTPS `PUBLIC_URL`. A reverse proxy must forward the entire site at the root, including the API, registry, token endpoint, webhooks, and Agent connections. Preserve streaming responses, WebSocket upgrades, and registry authentication headers. To terminate TLS in Dockerun, configure `TLS_CERT` and `TLS_KEY` and use the HTTPS health check. Use a private network for HTTP tests. Docker daemons and builders that access an HTTP registry also need the matching `insecure-registries` setting. Docker socket access gives Dockerun control over host Docker resources, so restrict administration to trusted users. [HTTPS setup](docs/DEPLOYMENT.md#https-反向代理) · [Security](docs/SECURITY.md)
- **Persistent data.** Compose uses the `dockerun-data` named volume at `/var/lib/dockerun`. It holds metadata, keys, built-in registry contents, and release material. For a host directory use `docker-compose.bind.yml` with the base Compose file and set `DATA_DIR` to an existing host directory. To use an existing MySQL instance, set `DATABASE_TYPE=mysql` and `MYSQL_DSN` and retain the persistent data mount. Existing MySQL and nginx-proxy deployments can use `deploy.sh`. [Storage](docs/DATA_STORAGE.md) · [MySQL](docs/MYSQL_SETUP.md) · [Deployment script](docs/DEPLOY_SCRIPT.md)
- **Offline console installation.** Build Dockerun for the target architecture on a connected machine and export the image with `docker save`. Transfer the archive with the Compose files, `.env.example`, `LICENSE`, and `NOTICE`. Load the image with `docker load`, set `DOCKERUN_IMAGE` to the loaded tag, and run `docker compose up -d --no-build --pull never`. Application images are prepared separately. [Offline installation](docs/DEPLOYMENT.md#离线安装)
- **Backups and upgrades.** Create and download a full backup from **Registry and Maintenance** before upgrading. Back up business data volumes, external registry contents, and installation settings separately. Keep the old image tag and restore data into a new, empty destination when needed. [Backup and restore](docs/DEPLOYMENT.md#备份与恢复) · [Upgrades](docs/DEPLOYMENT.md#升级与检查)

## First application deployment

1. Sign in as an administrator. Create an application and its first environment, or start in **Application Templates** (应用模板). Choose the local host, an Agent, or a fixed host group as supported by the template.
2. Prepare the images. Use image addresses, a configured registry, images already on the target, or offline release material, depending on the deployment type.
3. Open **Deploy Application** (发布应用). Review the Compose file or command, environment parameters, configuration files under `configs/`, scripts, and health checks.
4. Preview the release to verify the target and changes, then confirm to start deployment. Open **Deployments** (部署记录) to follow logs, per-host results, and health status.

For a complete Compose example and a walkthrough of authenticating, pushing images, and updating an environment, see [Quick start](docs/QUICKSTART.md#发布一个-compose-应用) and the [Registry guide](docs/REGISTRY_GUIDE.md).

## Local development

Use Go `1.26.0`, Node.js `22.17.0`, and pnpm `10.12.4`. Start Docker Desktop or Docker Engine, then run the Make targets from the repository root.

```bash
make install-deps
# Terminal 1
make dev-backend
# Terminal 2
make dev-frontend
```

The Vite UI is at `http://localhost:5999` and the backend listens on `127.0.0.1:8080`. Development data is stored in `data/local`. The Make targets default to `PUBLIC_URL=http://host.docker.internal:8080` for Docker Desktop. With Docker Engine on local Linux, run `make dev-backend PUBLIC_URL=http://localhost:8080`.

```bash
make build
make run
```

`make build` produces `bin/dockerun-server` and the `bin/dockerun` CLI. `make run` serves the built frontend at `http://localhost:8080`. To use the CLI against a Dockerun server, set `DOCKERUN_URL` and provide a scoped deployment token through `DOCKERUN_TOKEN`. It supports `deploy`, `package`, `import`, `promote`, `rollback`, and `status`. [Development guide](docs/DEVELOPMENT.md) · [CLI and Agents](docs/INTEGRATED_DEPLOYMENT.md#cli-与-agent)

## Documentation

The detailed guides are in Chinese.

| Guide | Covers |
| --- | --- |
| [Quick start](docs/QUICKSTART.md) | First startup, administrator setup, and a sample Compose deployment |
| [Deployment](docs/DEPLOYMENT.md) · [Configuration](docs/CONFIGURATION.md) | HTTPS, storage, environment variables, offline installation, backup, and upgrades |
| [Deployment workflow and API](docs/INTEGRATED_DEPLOYMENT.md) · [Template format](templates/README.md) | Releases, automation, offline bundles, CLI, Agents, and template packages |
| [Registry guide](docs/REGISTRY_GUIDE.md) · [Security](docs/SECURITY.md) | Built-in and external registries, accounts, tokens, and access boundaries |
| [Development](docs/DEVELOPMENT.md) · [Build and delivery](docs/BUILD_AND_DEPLOY.md) | Local development, checks, binaries, and multi-platform images |
| [Troubleshooting](docs/TROUBLESHOOTING.md) · [Verification](docs/VALIDATION.md) · [Documentation index](docs/README.md) | Common issues, verified behavior, and additional guides |

## License

Dockerun is licensed under the [Apache License, Version 2.0](LICENSE). The [NOTICE](NOTICE) retains the historical MIT notice and identifies the separately licensed Nacos schema.
