<div align="center">

<h1>PicBed Switcher</h1>

<p><b>Markdown image URL migration platform</b> — GitHub · Gitee · Tencent COS · Aliyun OSS · Qiniu · Baidu BOS · Huawei OBS · Upyun · MinIO · EasyImage</p>

<p><a href="README.md">简体中文</a> · <b>English</b></p>

Swapping image hosts means editing every Markdown document link by hand — one URL at a time.  
PicBed Switcher pulls that work into one **self-hosted** batch conversion platform: a Go backend, a Vue 3 console and PostgreSQL.  
One `docker compose up -d`, or a single binary downloaded from Releases, is all it takes.  
Your documents, conversion history and image-host credentials stay on your own disk and database; credentials are stored with AES-256-GCM encryption.

<p>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?labelColor=1f2937" alt="MIT License"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white&labelColor=1f2937" alt="Go 1.25+"></a>
  <a href="https://vuejs.org/"><img src="https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs&logoColor=white&labelColor=1f2937" alt="Vue 3"></a>
  <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/PostgreSQL-16%2B-4169E1?logo=postgresql&logoColor=white&labelColor=1f2937" alt="PostgreSQL 16+"></a>
</p>

<p>
  <b><a href="#preview">Preview</a></b> ·
  <a href="#what-it-does">What it does</a> ·
  <a href="#how-it-works">How it works</a> ·
  <a href="#tech-stack">Tech stack</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#deployment">Deployment</a> ·
  <a href="#supported-image-hosts">Hosts</a> ·
  <a href="#data-and-security">Security</a> ·
  <a href="#api-docs">API</a> ·
  <a href="#faq">FAQ</a> ·
  <a href="#repository-layout">Layout</a> ·
  <a href="#documentation">Docs</a>
</p>

</div>

---

## Preview

### Sign in

| Sign-in page |
| :---: |
| ![login](.github/images/picbed-switcher-login.jpg) |

### Workspace

| Dashboard |
| :---: |
| ![home](.github/images/picbed-switcher-home.jpg) |

## What it does

- **Multi-host adapters** — Ten image hosts out of the box: GitHub, Gitee, Tencent COS, Aliyun OSS, Qiniu, Baidu BOS, Huawei OBS, Upyun, MinIO and EasyImage, plus any self-hosted service with a compatible upload API.
- **Document conversion** — Upload a Markdown document, and the platform identifies every image URL and its host type automatically; pick a target host from your saved configs and batch-convert in one click, then download the converted document.
- **Local image upload** — Detects local image references in Markdown (relative paths, absolute paths, `<img>` tags), uploads the images to the target host and rewrites the URLs in place.
- **Conversion task queue** — Batch conversions and local uploads create persistent tasks; with Redis enabled a background worker consumes the queue at a configurable concurrency, and progress survives page refreshes by task ID. Without Redis it falls back to an in-process memory queue.
- **Object naming format** — Upload paths support variables such as `{y}`, `{m}`, `{d}`, `{hash}` and `{rand:N}`, so objects can be organized by date folder, content hash or any custom template.
- **Image host configuration** — Add, edit, delete and test host configs, with one default host; sensitive values are stored encrypted and masked in the console.
- **Conversion history** — Every conversion records source/target hosts, status and image counts; the detail view lists per-image replacement results and offers the converted document for download, with batch deletion for the list.
- **User authentication** — Registration, sign-in, email verification, password reset by email and JWT session management.

**It is not** an image host itself, nor a compression or CDN management tool: PicBed Switcher handles "batch migration and rewriting of image URLs in documents" — it does not proxy image traffic, cache images, or modify the originals.

The same host-migration job, done both ways:

| Situation | Manual rewriting | With PicBed Switcher |
| --- | --- | --- |
| Switching hosts | Edit every link in every document | Pick the target host, batch-convert once |
| New repo / new domain | Find-and-replace is error-prone | Host type and URLs are detected automatically |
| Local images | Upload by hand, then fix each link | Local upload rewrites URLs automatically |
| Progress | No idea how far you got | Persistent task queue, progress survives refreshes |
| Conversion records | Nothing to look back on | History keeps per-image replacement details |
| Host credentials | Scattered across scripts | Stored encrypted (AES-256-GCM) in the database |

## How it works

```text
        Browser
           │  http
           ▼
  ┌──────────────────────────────────────┐
  │  Nginx (in-container or on host)     │
  │  /                  → static assets  │
  │  /api/ · /swagger/  → Go backend     │
  └──────────────────────────────────────┘
        │                        │
        ▼                        ▼
  Vue 3 console            Go 1.25 backend
  Vite build output        Gin :8080
                                 │
                    ┌────────────┼────────────┐
                    ▼            ▼            ▼
              PostgreSQL    Upload adapters  Task queue
              users/configs  GitHub·COS·OSS… Redis (optional)
```

- **Who does what** — The console uploads documents, picks hosts and shows task progress and results; document parsing, URL detection, image upload, replacement and task scheduling all happen in the backend.
- **Where data lives** — PostgreSQL is the source of truth for users, host configs, conversion tasks and history; image-host credentials are stored encrypted with AES-256-GCM.
- **Task queue** — With Redis enabled, conversion tasks are written to a Redis List and consumed by a background worker; without it the queue falls back to process memory, and tasks still queued are re-enqueued automatically at startup.
- **Public surface** — Registration, sign-in, email verification, password reset and health check are anonymous; every other endpoint requires `Authorization: Bearer <token>`.

## Tech stack

| Layer | Choice |
| --- | --- |
| Backend language | Go 1.25+ |
| HTTP framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL 16+ ([GORM](https://gorm.io/)) |
| Task queue | Redis 5.0+ (optional; falls back to in-process memory queue) |
| Auth & crypto | JWT, bcrypt password hashing, AES-256-GCM credential encryption |
| API docs | [swag](https://github.com/swaggo/swag) + Swagger UI |
| Frontend framework | Vue 3 + TypeScript |
| Build tool | [Vite](https://vite.dev/) |
| UI & state | Element Plus, Pinia, Vue Router, Lucide |
| Object storage client | [MinIO Go Client](https://github.com/minio/minio-go) (S3-compatible) |
| Runtime packaging | Docker (Nginx + Supervisor multi-process) |

## Quick start

Local development requires **Go 1.25+**, **Node.js 20+** and **PostgreSQL 16+**.

```bash
git clone https://github.com/zyx3721/picbed-switcher.git
cd picbed-switcher
```

**Backend**

```bash
cd backend
go mod download
cp .env.example .env       # adjust database connection and secrets as needed
go run cmd/main.go
```

The backend runs at `http://localhost:8080` by default; on first start it runs the database migrations and creates the default admin `admin / 123456` (**change the password right after signing in**).

**Frontend** (in another terminal)

```bash
cd frontend
npm install
npm run dev
```

The frontend runs at `http://localhost:5173` by default. If the backend does not run on port 8080, create `frontend/.env` with `VITE_API_BASE_URL=http://localhost:<backend-port>`.

Open `http://localhost:5173` to sign in; Swagger lives at `http://localhost:8080/swagger/index.html`.

## Deployment

Only two paths are kept: **Docker Compose** (recommended) and **Release binaries**. Source builds, host Nginx reverse proxy and HTTPS are covered in full in the [full manual](docs/manual.md) (Chinese).

### Option 1: Docker Compose

The image bundles the Go backend, Nginx and the built frontend, managed by Supervisor with only port 80 exposed, plus a PostgreSQL container:

```bash
cd deploy
cp .env.example .env
vim .env                   # at least change DB_PASSWORD and JWT_SECRET
docker compose up -d
```

Service management:

```bash
docker compose ps                       # service status
docker compose logs -f picbed-switcher  # live logs
docker compose restart picbed-switcher  # restart the app
docker compose down                     # stop everything
```

**Access**

- Console: `http://your-domain.com`, default account `admin / 123456`
- API docs: `http://your-domain.com/swagger/index.html`
- Health check: `http://your-domain.com/health`

For custom domains, HTTPS or an existing PostgreSQL instance, see [chapter 3 of the full manual](docs/manual.md).

### Option 2: Release binaries

Download the matching archive from the [GitHub Releases](https://github.com/zyx3721/picbed-switcher/releases) page, then verify, extract, configure and start as below. Binary deployment requires your own PostgreSQL 16+ database.

**Which package to download**

| Your machine | File |
| --- | --- |
| Linux x86_64 | `picbed-switcher_<version>_linux_amd64.tar.gz` |
| Linux ARM64 (Kunpeng, Phytium, etc.) | `picbed-switcher_<version>_linux_arm64.tar.gz` |
| macOS (Intel) | `picbed-switcher_<version>_darwin_amd64.tar.gz` |
| macOS (Apple silicon) | `picbed-switcher_<version>_darwin_arm64.tar.gz` |
| Windows x86_64 | `picbed-switcher_<version>_windows_amd64.zip` |
| Windows ARM64 | `picbed-switcher_<version>_windows_arm64.zip` |
| Console (required on any platform) | `picbed-switcher-frontend_<version>.tar.gz` |
| Checksums | `SHA256SUMS` |

The backend archive contains the `picbed-switcher` binary (`picbed-switcher.exe` on Windows), `.env.example` and `README.txt`; the frontend archive contains the Vite static build, ready for Nginx or any static server.

**1. Verify and extract**

```bash
VERSION=3.0.1
mkdir -p /data/picbed-switcher && cd /data/picbed-switcher
sha256sum -c SHA256SUMS
mkdir -p backend frontend
tar -xzf picbed-switcher_${VERSION}_linux_amd64.tar.gz -C backend --strip-components=1
tar -xzf picbed-switcher-frontend_${VERSION}.tar.gz -C frontend
```

**2. Configure and start the backend**

```bash
cd /data/picbed-switcher/backend
cp .env.example .env
vim .env                   # database connection, JWT_SECRET and SMTP mail
./picbed-switcher
```

The backend listens on `http://localhost:8080` by default; on first start it runs the database migrations and creates the default admin.

The backend also accepts command-line flags; an explicit flag takes precedence over environment variables and the `.env` file. `./picbed-switcher -v` prints version info (version, commit, build date) and `./picbed-switcher -h` lists all flags:

| Flag | Equivalent env var | Description |
| --- | --- | --- |
| `-host` | `SERVER_HOST` | Listen host |
| `-port` | `SERVER_PORT` | Listen port |
| `-mode` | `GIN_MODE` | Gin mode (debug/release/test) |
| `-db-host` | `DB_HOST` | Database host |
| `-db-port` | `DB_PORT` | Database port |
| `-db-name` | `DB_NAME` | Database name |
| `-db-user` | `DB_USER` | Database user |
| `-db-password` | `DB_PASSWORD` | Database password |
| `-db-sslmode` | `DB_SSLMODE` | Database SSL mode |
| `-jwt-secret` | `JWT_SECRET` | JWT signing secret |
| `-env` | — | Path to the `.env` config file |
| `-v`, `-version` | — | Print version info and exit |

For a long-running setup, hand it to systemd:

```ini
# /etc/systemd/system/picbed-backend.service
[Unit]
Description=PicBed Switcher Backend
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/data/picbed-switcher/backend
ExecStart=/data/picbed-switcher/backend/picbed-switcher
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload && systemctl enable --now picbed-backend
```

**3. Deploy the frontend behind Nginx**

Serve the `frontend/` directory as the Nginx static root and proxy `/api/` to the backend:

```nginx
server {
    listen 80;
    server_name your-domain.com;
    client_max_body_size 50m;

    location / {
        root /data/picbed-switcher/frontend;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;
    }

    location /swagger/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
    }

    location = /health {
        proxy_pass http://127.0.0.1:8080/health;
    }
}
```

The full HTTPS example with 80→443 redirect is in [chapter 4 of the full manual](docs/manual.md) (Chinese).

**4. Access**

Same as the Docker path: console at `http://your-domain.com` (`admin / 123456`), API docs at `/swagger/index.html`, health check at `/health`.

## Supported image hosts

| Host | Required config |
| --- | --- |
| GitHub | Personal Access Token, repository |
| Gitee | Private Token, repository |
| Tencent COS | SecretId, SecretKey, bucket |
| Aliyun OSS | AccessKeyId, AccessKeySecret, bucket |
| Qiniu | AccessKey, SecretKey, bucket |
| Baidu BOS | AccessKeyId, SecretAccessKey, bucket, region |
| Huawei OBS | AccessKeyId, SecretAccessKey, bucket, region |
| Upyun | Service name, operator, password, acceleration domain |
| MinIO | Endpoint, AccessKey, SecretKey, bucket (optional region, SSL, public domain) |
| EasyImage | API URL, Token |
| Other hosts | Any compatible upload API URL, Token |

All except EasyImage support a custom object naming format; available variables and examples are in [section 6.2.1 of the full manual](docs/manual.md) (Chinese).

## Data and security

```text
Console account signs in, manages configs, converts documents
        +
Per-endpoint Bearer Token checks; anonymous surface limited to sign-up/sign-in and password recovery
        +
Image-host tokens / secrets stored with AES-256-GCM, masked in the console
        +
Passwords hashed with bcrypt; rate limiting on public endpoints
```

- **Change the default password first** — Update the default `admin` password immediately after the first deployment.
- **Rotate the JWT secret** — Always set `JWT_SECRET` to a long random string in production, never the sample value; resetting it invalidates every issued token.
- **Enable HTTPS** — Terminate TLS at Nginx in production; see [section 4.4.2 of the full manual](docs/manual.md) (Chinese).
- **Back up the database regularly** — PostgreSQL holds all business data; schedule periodic `pg_dump` backups.

## API docs

The backend ships with Swagger/OpenAPI, available as soon as the service starts:

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Health check**: `GET /health`

The only anonymous endpoints are registration, sign-in, email verification, password reset and the health check; every other endpoint requires `Authorization: Bearer <token>`.

The full endpoint list grouped by module (auth, image host configs, document conversion) is in [chapter 5 of the full manual](docs/manual.md) (Chinese).

After changing an endpoint, regenerate the Swagger artifacts in `backend/`:

```bash
swag init -g cmd/main.go -o docs
```

## FAQ

**A conversion failed — what now?**

Check in order: the host config, whether the token/keys are still valid, network connectivity, and whether the host API rate-limited you; the error summary in the conversion history detail gives the specific reason.

**How do I back up data?**

```bash
docker exec picbed-postgres pg_dump -U your_user picbed_switcher > backup.sql
```

**How do I restore data?**

```bash
docker exec -i picbed-postgres psql -U your_user picbed_switcher < backup.sql
```

**Which local image references does local upload support?**

Relative paths (`./images/a.png`, `images/a.png`), Windows absolute paths and `<img src="...">` tags; `http://`/`https://` remote URLs are left untouched — use document conversion for those. More questions are answered in the [full manual](docs/manual.md) (Chinese).

## Repository layout

```text
picbed-switcher/
├── backend/                 Go backend
│   ├── cmd/                 application entrypoint
│   ├── internal/            internal modules
│   │   ├── buildinfo/       build version info (injected via -ldflags)
│   │   ├── config/          environment config loading
│   │   ├── database/        database connection and initialization
│   │   ├── handler/         HTTP routes and handlers
│   │   ├── middleware/      auth, CORS, rate-limit middlewares
│   │   ├── model/           GORM data models
│   │   ├── picbed/          image host upload adapters
│   │   └── utils/           crypto, JWT and Markdown utilities
│   ├── migrations/          PostgreSQL migrations
│   ├── docs/                generated Swagger artifacts
│   └── .env.example         environment template
├── frontend/                Vue 3 console
│   ├── public/              static assets
│   ├── src/
│   │   ├── components/      pages and dialog components
│   │   ├── composables/     business state, requests and form logic
│   │   │   └── workspace/   workspace logic: hosts, conversion, local upload
│   │   ├── App.vue          root component
│   │   ├── main.ts          frontend entry
│   │   └── style.css        global styles
│   └── .env.example         frontend environment template
├── deploy/                  Docker build and deployment files
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── nginx.conf
│   ├── supervisord.conf
│   ├── entrypoint.sh
│   └── .env.example
├── docs/                    full manual
│   └── manual.md
├── verchanglog/             release changelogs
├── .github/                 GitHub Actions workflows and preview images
├── LICENSE
├── README.md                简体中文
└── README.en.md             English (this file)
```

## Documentation

| Start here | Then |
| --- | --- |
| [Quick start](#quick-start) | Run the backend and frontend locally; default account and ports |
| [Deployment](#deployment) | Docker Compose and release binaries |
| [Full manual](docs/manual.md) | Local development, deployment details, Nginx and HTTPS examples, API list, usage guide (Chinese) |
| [简体中文 README](README.md) | The same content in Chinese |

## Releases

| Version | Date | Changelog |
| --- | --- | --- |
| v3.0.1 | 2026-05-19 | [verchanglog/v3.0.1.md](verchanglog/v3.0.1.md) (Chinese) |
| v3.0.0 | 2026-05-19 | [verchanglog/v3.0.0.md](verchanglog/v3.0.0.md) (Chinese) |
| v2.1.0 | 2026-05-18 | [verchanglog/v2.1.0.md](verchanglog/v2.1.0.md) (Chinese) |
| v2.0.0 | 2026-05-18 | [verchanglog/v2.0.0.md](verchanglog/v2.0.0.md) (Chinese) |
| v1.0.0 | 2026-05-18 | [verchanglog/v1.0.0.md](verchanglog/v1.0.0.md) (Chinese) |

Build assets and release notes for every version live on [GitHub Releases](https://github.com/zyx3721/picbed-switcher/releases).

## Acknowledgements

Thanks to these open-source projects and communities:

- [Gin](https://github.com/gin-gonic/gin) — high-performance Go web framework
- [GORM](https://gorm.io/) — Go ORM library
- [PostgreSQL](https://www.postgresql.org/) — reliable open-source relational database
- [Vue](https://vuejs.org/) — progressive JavaScript framework
- [Vite](https://vite.dev/) — fast frontend build tool
- [Element Plus](https://element-plus.org/) — Vue 3 component library
- [Lucide](https://lucide.dev/) — clean, consistent icon library
- [MinIO Go Client](https://github.com/minio/minio-go) — S3-compatible object storage client
- [EasyImage](https://github.com/icret/EasyImages2.0) — simple self-hosted image hosting

And to everyone who has contributed code, suggestions and bug reports.

## License

Released under the [MIT License](LICENSE): use, copy, modify, merge, publish, distribute, sublicense and sell freely, as long as the copyright and licence notice are kept in all copies or substantial portions.

## Contact

- **Email**: 416685476@qq.com
- **GitHub Issues**: [zyx3721/picbed-switcher/issues](https://github.com/zyx3721/picbed-switcher/issues)
- **Project home**: [github.com/zyx3721/picbed-switcher](https://github.com/zyx3721/picbed-switcher)

---

**⭐ If this project helps you, a star is appreciated!**
