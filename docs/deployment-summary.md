# Backend Deployment — Session Summary (2026-05-26)

Deployed a production Go backend to AWS EC2 with HTTPS, terminating at `https://api.prakhargaming.com`.

## Final architecture

```
Client (HTTPS)
   │
   ▼
Caddy 2.8.4 (TLS termination, auto-HTTPS via Let's Encrypt, gzip, HTTP/2 + HTTP/3)
   │  reverse_proxy → localhost:8080
   ▼
Go backend (systemd-managed service, runs as `webapp` user)
   │
   ├── MongoDB Atlas (vector DB for embeddings + blog content)
   ├── Google Gemini (chat + embeddings)
   └── Clerk (optional JWT auth on /send-message)
```

- **Host:** t3.micro, Amazon Linux 2023, Elastic IP `3.15.54.181`
- **DNS:** `api.prakhargaming.com` → EIP
- **Endpoints live:** `GET /blogs`, `POST /send-message`

## What was built / configured

### Application-layer (Go)
- **Optional Clerk JWT middleware** on `/send-message` — anonymous requests allowed; if a valid `Authorization: Bearer <token>` is present, user ID is attached to request context
- **Per-key token-bucket rate limiter** on `/send-message`
  - Anonymous (by IP): 5/hour, burst 5
  - Authenticated (by Clerk user ID): 30/hour, burst 10
  - Client IP derived from rightmost `X-Forwarded-For` entry (trusting single upstream proxy)
- **Tight CORS allowlist:** `https://prakhargaming.com` and `http://localhost:3000`, with `Vary: Origin`
- **Config promoted to fail-fast:** `CLERK_SECRET_KEY` required at startup via `mustGetEnv`

### Infrastructure
- **systemd unit** with hardening: `ProtectSystem=strict`, `ReadOnlyPaths=/opt/prakhar-backend`, `NoNewPrivileges`, `PrivateTmp`, dedicated `webapp` system user, no shell
- **EnvironmentFile** at `/etc/prakhar-backend.env`, mode `640`, owned `root:webapp`
- **Caddy** as TLS-terminating reverse proxy
  - Installed via static binary (Caddy COPR repo doesn't support AL2023)
  - Official systemd unit from `caddyserver/dist`
  - Auto-HTTPS via Let's Encrypt (tls-alpn-01 challenge succeeded)
- **AWS Security Group:** SSH (22) restricted to home IP, HTTP/HTTPS (80/443) open to world
- **MongoDB Atlas Network Access:** EIP allowlisted

## Debugging wins this session

1. **SSH connection timeout** — home ISP rotated public IP; SG inbound rule for SSH was still pinned to the old `/32`. Updated rule to current IP.
2. **systemd service crash loop** — `failed to read system prompt at /home/prakhargaming/Dev/...` — the `.env` file pushed to the box carried the developer's local `SYSTEM_PROMPT_PATH`. Fixed with `sed -i` to point at the deploy location.
3. **Caddy package unavailable on AL2023** — COPR repo `@caddy/caddy` has Fedora/EPEL targets only. Fell back to the official static binary release.

## What's *not* in scope (deferred)
- Clerk webhook handler (SendGrid welcome email) — handler exists in `adapters/webhook.go` but not registered in `main.go`; SendGrid account expired
- Horizontal scaling — rate limiter is in-memory, single-instance only. Would need Redis if scaled out
- CI/CD — deploys are manual SCP + `systemctl restart`
- Monitoring/alerting — relying on journald only
- Health-check endpoint (`GET /health`) — not yet added

## Tech stack (resume-friendly bullet list)
- **Go 1.25** (stdlib `net/http`, no framework)
- **AWS EC2** (Amazon Linux 2023, t3.micro, Elastic IP, Security Groups, VPC)
- **Caddy** (reverse proxy, automatic Let's Encrypt TLS via ACME)
- **systemd** (service unit with sandboxing hardening, EnvironmentFile)
- **MongoDB Atlas** (vector search)
- **Clerk** (JWT auth)
- **Google Gemini** (LLM + embeddings)
