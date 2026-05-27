# Handoff: Backend Deployment to AWS EC2

Status as of 2026-05-26. The user is mid-deployment, stuck on creating the env file on the EC2 box.

## TL;DR for the next agent

The Go backend code work is complete and builds cleanly. The user is currently SSH'd into a fresh Amazon Linux 2023 EC2 instance and partway through the deploy steps. The `webapp` user, `/opt/prakhar-backend/` directories, env file, and the systemd + Caddy services are NOT yet set up on the box. The user got tripped up by pasted multi-line commands fragmenting in their terminal (ghostty + zsh quirks).

**Your immediate job**: walk the user through the remaining EC2 setup commands, one at a time, and debug whatever breaks.

## Project context

- **Repo**: `/home/prakhargaming/Dev/prakhar-website-backend` on the user's Fedora dev box.
- **Stack**: Go 1.25, `net/http` (no framework), MongoDB Atlas, Google Gemini (chat + embeddings), Clerk (auth).
- **Two public endpoints**: `GET /blogs`, `POST /send-message`.
- **User profile** (from memory): experienced Python/FastAPI dev learning Go. Explain Go concepts via FastAPI analogues when relevant.

## What this session accomplished

The original punch list for shipping to EC2:

- [x] Clerk JWT middleware on `/send-message` (optional auth — anon allowed, user ID attached to ctx if token present)
- [x] Tight CORS (origin allowlist + `Vary: Origin`)
- [x] Per-user / per-IP rate limit on `/send-message` (in-memory token bucket)
- [x] systemd unit + EnvironmentFile (artifact written; not yet installed on EC2)
- [x] Caddy for TLS (Caddyfile written; not yet installed on EC2)
- [x] Elastic IP + Atlas allowlist + DNS (done in respective consoles)
- [~] **Service running on EC2** ← in progress, this is where you pick up

The Clerk *webhook* route (SendGrid welcome email) was intentionally skipped — the user's SendGrid account expired and it's out of scope. The handler still exists in `adapters/webhook.go` but is never registered in `main.go`. The two env vars it would need (`CLERK_WEBHOOK_SECRET_PROD`, `SENDGRID_API_KEY`) were downgraded from `mustGetEnv` to `getEnv` so the app starts without them.

## Files added / changed this session

```
middleware/auth.go            NEW — OptionalClerkAuth + UserID(ctx) helpers
middleware/ratelimit.go       NEW — RateLimiter struct, token bucket per key
deploy/prakhar-backend.service NEW — systemd unit
deploy/Caddyfile              NEW — reverse proxy + auto-HTTPS
deploy/prakhar-backend.env.example NEW — template, real values NOT committed
config/config.go              MODIFIED — added ClerkSecretKey (required);
                              demoted ClerkWebhookSecret + SendgridAPIKey to optional
main.go                       MODIFIED — clerk.SetKey, OptionalClerkAuth wraps
                              rateLimiter.Middleware wraps SendMessage; CORS
                              allowlist + Vary: Origin; Authorization header allowed
.gitignore                    MODIFIED — added `deploy/*.env` and `server`
go.mod / go.sum               MODIFIED — added clerk-sdk-go/v2, golang.org/x/time/rate
```

`server` is the cross-compiled binary (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o server .`). It was SCP'd to the EC2 box but may or may not still be in `~ec2-user/` — the user's terminal output is ambiguous.

## Key design decisions (don't undo these without asking)

1. **Optional auth, not strict.** `/send-message` accepts anonymous traffic. The user explicitly chose this so visitors can chat without signing up. The rate limiter does the actual protection — anon: 5/hour burst 5, authed: 30/hour burst 10. If you ever switch to strict auth, drop the anon path in `middleware/ratelimit.go:classify`.

2. **In-memory rate limiter, single instance.** State resets on restart and isn't shared across instances. Fine for a single t3.micro. If the user ever wants horizontal scaling, swap to Redis — but don't pre-emptively.

3. **Rightmost X-Forwarded-For for client IP.** Assumes a single trusted reverse proxy (Caddy) in front. The proxy *appends* to client-sent XFF, so the rightmost entry is the real IP that Caddy added. If the Go service is ever exposed directly (no Caddy in front), an attacker can spoof XFF to bypass anon limits. The systemd hardening + SG rules should prevent that, but it's worth knowing.

4. **CORS allowlist hardcoded in `main.go`**, not config-driven. Two origins: `https://prakhargaming.com` and `http://localhost:3000`. The user said this is fine; resist moving it to config unless they ask.

5. **systemd hardening on `prakhar-backend.service`** uses `ProtectSystem=strict`, `ReadOnlyPaths=/opt/prakhar-backend`, `NoNewPrivileges`, etc. Don't loosen these — the binary doesn't need write access anywhere.

## Deployment state on EC2

Confirmed done (in respective consoles):
- EC2 instance launched (Amazon Linux 2023, `t3.micro` likely)
- Security group: SSH 22 from user's IP, HTTP 80 + HTTPS 443 from 0.0.0.0/0
- Elastic IP `3.15.54.181` allocated **and now associated** with the instance (user initially forgot to associate it — the instance was reachable at the auto-assigned `3.18.212.197` until they fixed this)
- MongoDB Atlas allowlist updated with the EIP
- DNS: `api.prakhargaming.com` CNAME → resolves to `3.15.54.181`

Confirmed done on the EC2 box:
- Artifacts SCP'd to `~ec2-user/`: `server`, `system_prompt.txt`, `prakhar-backend.service`, `Caddyfile` (per the user's last clean `ls` output)
- `nmap-ncat` and `bind-utils` installed during diagnostics

NOT yet done on the EC2 box (the immediate work):
- `webapp` system user
- `/opt/prakhar-backend/` directory tree
- Binary + prompt + service file moved into place
- `/etc/prakhar-backend.env` (last attempt failed — indented EOF terminator and truncated first line)
- `systemctl daemon-reload && systemctl enable --now prakhar-backend`
- Caddy installation
- `/etc/caddy/Caddyfile` placement
- `systemctl enable --now caddy`
- End-to-end smoke test: `curl https://api.prakhargaming.com/blogs`

## Things that broke / are about to break

1. **The user's terminal (ghostty) splits multi-line pasted commands into separate shell invocations.** When they pasted `sudo useradd --system --no-create-home --shell \n  /usr/sbin/nologin webapp` as a block, the newline broke it. Have them run one command at a time, or use bracketed-paste mode. Don't paste multi-command blocks.

2. **The env-file heredoc failure mode is subtle.** Their previous attempt indented the closing `EOF` (`>   EOF`), so bash never matched it as the terminator. Heredoc terminators must be flush-left unless you use `<<-EOF` (tabs only, not spaces). Tell them to disable auto-indent when pasting.

3. **TERM=xterm-ghostty is not in the EC2 termcap database.** `nano` and `vim` will fail with "cannot initialize terminal type". Workaround: `TERM=xterm nano file`, or just use `tee` heredoc as we've been doing. They don't need an editor for any of this.

4. **DNS already resolves to the EIP, so once Caddy starts, ACME will try immediately.** If anything is misconfigured (security group blocking 80, Caddyfile wrong, etc.), Let's Encrypt will rate-limit failed challenges. Don't `systemctl restart caddy` repeatedly in a loop — read the logs between attempts.

5. **MongoDB cluster hostname has no A record** — only SRV. So `nc prakhar-cluster.houdk.mongodb.net 27017` fails with "Name or service not known", but this is normal. The actual reachable hosts are the shards (e.g. `prakhar-cluster-shard-00-00.houdk.mongodb.net`). Don't let the user mistake this for an Atlas problem.

## Credentials — needs user attention

The user's `.env` (and now `/etc/prakhar-backend.env` once it's created) contains live production credentials. These were exposed in this conversation when the user asked Claude to read the `.env` file. As of 2026-05-26 **none have been confirmed rotated**:

- **Clerk `sk_live_...` key** ← highest priority to rotate; grants full backend access to the Clerk app
- MongoDB Atlas password (user: `prakhar-fedora`)
- Gemini API key (lower stakes — rate-limited)

If the user hasn't rotated, gently nudge again before they start the service in prod. Once they do rotate, the new values must go in both the local `.env` and `/etc/prakhar-backend.env` on EC2.

## Recommended order of operations from here

Assuming the user is still SSH'd into the EC2 box at `ec2-user@3.15.54.181`:

1. **Diagnose current state:**
   ```bash
   ls ~
   ls /opt/prakhar-backend/ 2>/dev/null || echo "(no /opt/prakhar-backend)"
   id webapp 2>/dev/null || echo "(no webapp user)"
   sudo test -f /etc/prakhar-backend.env && echo "env file exists" || echo "(no env file)"
   sudo systemctl is-active prakhar-backend 2>/dev/null
   ```
   This tells you exactly where they're stuck.

2. **If `server` binary is missing from both `~` and `/opt/prakhar-backend/`**, have them re-SCP from the dev box:
   ```bash
   # on dev box
   scp -C -i /home/prakhargaming/Downloads/da-key-pair.pem server data/system_prompt.txt deploy/prakhar-backend.service deploy/Caddyfile ec2-user@3.15.54.181:~/
   ```
   Use `-C` (compression) — the user's home internet is slow and the 14MB binary stalled at 5 KB/s without it.

3. **Run setup commands one at a time** (don't paste blocks):
   ```bash
   sudo useradd --system --no-create-home --shell /usr/sbin/nologin webapp
   sudo mkdir -p /opt/prakhar-backend/data
   sudo mv ~/server /opt/prakhar-backend/server
   sudo mv ~/system_prompt.txt /opt/prakhar-backend/data/
   sudo mv ~/prakhar-backend.service /etc/systemd/system/
   sudo chown -R webapp:webapp /opt/prakhar-backend
   sudo chmod +x /opt/prakhar-backend/server
   ```

4. **Env file** — paste as one block, EOF flush-left, no leading whitespace anywhere:
   ```bash
   sudo tee /etc/prakhar-backend.env > /dev/null <<'EOF'
   MONGODB_URI=mongodb+srv://prakhar-fedora:<PASSWORD>@prakhar-cluster.houdk.mongodb.net/?appName=prakhar-cluster
   MONGODB_VECTOR_DATABASE=Prakharbase
   MONGODB_VECTOR_COLLECTION=vector_database
   GEMINI_API_KEY=<KEY>
   CLERK_SECRET_KEY=<KEY>
   PORT=8080
   SYSTEM_PROMPT_PATH=/opt/prakhar-backend/data/system_prompt.txt
   EOF
   sudo chown root:webapp /etc/prakhar-backend.env
   sudo chmod 640 /etc/prakhar-backend.env
   ```
   The user has the actual secret values in their local `.env`. If they've rotated, use the new values.

5. **Start the Go service and verify Mongo connects:**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now prakhar-backend
   sudo journalctl -u prakhar-backend -e
   ```
   Look for `listening on :8080` with no errors. If you see a Mongo timeout, the Atlas allowlist is wrong (verify `3.15.54.181/32` is in Atlas Network Access).

6. **Install Caddy (Amazon Linux 2023):**
   ```bash
   sudo dnf install -y 'dnf-command(copr)'
   sudo dnf copr enable -y @caddy/caddy
   sudo dnf install -y caddy
   ```
   If COPR fails on AL2023 (it sometimes does), fall back to the static binary:
   ```bash
   ARCH=amd64
   VERSION=2.8.4
   curl -L "https://github.com/caddyserver/caddy/releases/download/v${VERSION}/caddy_${VERSION}_linux_${ARCH}.tar.gz" | sudo tar -xz -C /usr/local/bin caddy
   sudo chmod +x /usr/local/bin/caddy
   ```
   The static-binary path also needs a systemd unit — grab the official one from https://github.com/caddyserver/dist/blob/master/init/caddy.service.

7. **Drop in Caddyfile, start Caddy, watch ACME:**
   ```bash
   sudo mv ~/Caddyfile /etc/caddy/Caddyfile
   sudo systemctl enable --now caddy
   sudo journalctl -u caddy -f
   ```
   Look for `certificate obtained successfully` within ~30s. If ACME fails: DNS not propagated, port 80 blocked, or Squarespace doing some proxying we didn't expect.

8. **Smoke test from the user's laptop:**
   ```bash
   curl -i https://api.prakhargaming.com/blogs
   ```
   Expected: 200 with blogs JSON. If TLS fails, Caddy hasn't got the cert yet.

## After the service is up

These are not in scope for this session but worth knowing:
- No monitoring / alerting set up. CloudWatch agent or just journald → nothing for now.
- No backup / log rotation strategy. journald handles its own rotation by default.
- No CI/CD. Future deploys are manual SCP + `systemctl restart`.
- The frontend will need to be updated to call `https://api.prakhargaming.com` and to pass the Clerk session token as `Authorization: Bearer <token>` for `/send-message` (when the user wants authed users to get the higher rate limit).

## Open questions to ask the user

- Have you rotated the Clerk secret key, Mongo password, and Gemini key since they appeared in conversation? (See "Credentials" section.)
- Is the frontend already configured to send `Authorization: Bearer <Clerk token>` for `/send-message`, or is that the next thing to wire up after the backend is live?
- Do you want me to add a basic health-check endpoint (`GET /health`) for monitoring / load balancers? Not in scope here but the natural next step.
