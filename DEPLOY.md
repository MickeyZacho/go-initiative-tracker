# Deploying the initiative tracker (self-hosted + Cloudflare Tunnel)

This deploys the whole stack (Go backend, Postgres, React frontend) on a single
machine you control, reached at your simply.com domain through a **Cloudflare
Tunnel**. No router port-forwarding, no exposed home IP, works behind CGNAT, and
Cloudflare provides HTTPS for free.

```
internet ──► Cloudflare edge (TLS) ──► cloudflared ──► caddy
                                                        ├─ /api/* ─► backend ─► db
                                                        └─ /*     ─► frontend
```

You'll run it on your **Windows desktop** first, then optionally move the exact
same setup to a **Raspberry Pi** later (see the last section) — the migration is
just copying files and running one command, because nothing depends on the host.

The files that make this work:

| File | Purpose |
|------|---------|
| `docker-compose.prod.yml` | The full stack: db, backend, frontend, caddy, cloudflared |
| `caddy/` | Reverse proxy image (Dockerfile + Caddyfile): serves the frontend, strips `/api` → backend |
| `.env.prod.example` | Template for the secrets/URLs you fill in |

---

## Prerequisites

- **Docker Desktop** for Windows installed and running.
- A **Cloudflare account** (free) — https://dash.cloudflare.com/sign-up
- Your **domain** registered at simply.com.
- A **Discord application** (for login) — https://discord.com/developers/applications

---

## Step 1 — Move your domain's DNS to Cloudflare

The domain stays *registered* at simply.com; Cloudflare just becomes its DNS
provider (required for the tunnel).

1. In the Cloudflare dashboard: **Add a site** → enter your domain → pick the
   **Free** plan.
2. Cloudflare shows you **two nameservers** (e.g. `xxx.ns.cloudflare.com`).
3. Log in to **simply.com** → your domain → **DNS / Nameservers** settings →
   replace simply.com's nameservers with the two from Cloudflare.
4. Wait for Cloudflare to show the domain as **Active** (usually minutes, can be
   up to a few hours). You don't need to create any A/AAAA records yourself — the
   tunnel adds a CNAME for you in Step 3.

---

## Step 2 — Configure the Discord application

1. In the Discord Developer Portal → your app → **OAuth2**.
2. Under **Redirects**, add exactly:
   ```
   https://YOUR-DOMAIN/api/auth/discord/callback
   ```
   (The `/api` prefix matters — Caddy strips it before the backend sees the route.)
3. Copy the **Client ID** and **Client Secret** for the `.env` in Step 4.

---

## Step 3 — Create the Cloudflare Tunnel

1. Cloudflare dashboard → **Zero Trust** → **Networks** → **Tunnels** →
   **Create a tunnel** → choose **Cloudflared** → name it (e.g. `initiative`).
2. On the install screen, copy the **tunnel token** (the long string in the
   `cloudflared ... run <TOKEN>` command). That's your `TUNNEL_TOKEN`.
3. Add a **Public Hostname**:
   - **Subdomain**: blank (for the root domain) or e.g. `app`
   - **Domain**: your domain
   - **Type**: `HTTP`
   - **URL**: `caddy:80`
   
   This CNAME is created automatically. Save.

> The service is `caddy:80` because cloudflared and caddy share the compose
> network — caddy then routes to the frontend/backend containers.

---

## Step 4 — Fill in the environment file

From the project root:

```powershell
Copy-Item .env.prod.example .env
```

Edit `.env` and set every value. Generate the secrets (in the Bash tool, Git
Bash, or WSL — `openssl` ships with Git for Windows):

```bash
openssl rand -hex 24   # -> DB_PASSWORD
openssl rand -hex 32   # -> SESSION_SECRET
```

Set `FRONTEND_URL=https://YOUR-DOMAIN`, the Discord client id/secret, the
matching `DISCORD_REDIRECT_URL`, and the `TUNNEL_TOKEN` from Step 3.

---

## Step 5 — Launch

```powershell
docker compose -f docker-compose.prod.yml up -d --build
```

Check everything is healthy:

```powershell
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f cloudflared   # look for "Registered tunnel connection"
```

Then open **https://YOUR-DOMAIN**. Click **Login with Discord** to verify the
full OAuth round-trip.

> Local smoke test: http://localhost:8080 serves the frontend directly, but
> **login won't work there** — the auth cookies are marked `Secure` and only
> travel over HTTPS, so log in through the real `https://` domain.

### Everyday commands

```powershell
docker compose -f docker-compose.prod.yml logs -f backend    # tail backend logs
docker compose -f docker-compose.prod.yml down               # stop (keeps the db volume)
docker compose -f docker-compose.prod.yml up -d --build       # apply code changes
```

Data lives in the `db_data` Docker volume and survives restarts. Schema changes
apply automatically at startup via the embedded goose migrations.

---

## Optional — DietPi dashboard on its own subdomain

Exposes the Pi's **DietPi dashboard** (system panel on host port `5252`) at
`dietpi.YOUR-DOMAIN`, routed through the same tunnel and Caddy. The dashboard
runs on the *host*, not in a container, so Caddy reaches it through the docker
host gateway (already wired: the `caddy` service has
`extra_hosts: host.docker.internal:host-gateway`, and the Caddyfile proxies the
`DIETPI_HOSTNAME` host to `host.docker.internal:5252`).

> **Security first.** This is a full system-admin panel (reboot, service
> control, terminal). Do **not** leave it protected only by its own password —
> gate the subdomain with **Cloudflare Access** (step 3 below), which puts an
> identity check in front of it for free.

1. **Set the hostname** in your prod `.env`:
   ```
   DIETPI_HOSTNAME=dietpi.YOUR-DOMAIN
   ```
   (Leave it unset/blank to disable the route — the Caddyfile falls back to a
   host nobody uses, so the main app is unaffected.)

2. **Add a Public Hostname** to the tunnel (Zero Trust → Networks → Tunnels →
   your tunnel → **Public Hostname** → *Add*):
   - **Subdomain**: `dietpi`
   - **Domain**: your domain
   - **Type**: `HTTP`
   - **URL**: `caddy:80`  *(same as the main app — Caddy routes by Host header)*

3. **Protect it with Cloudflare Access** (Zero Trust → Access → **Applications**
   → *Add an application* → **Self-hosted**):
   - **Application domain**: `dietpi.YOUR-DOMAIN`
   - Add a **policy** allowing only your email (Action: *Allow*, Include: *Emails*
     → your address). Now Cloudflare prompts for login before anyone reaches the
     dashboard.

4. **Apply** — rebuild so the new Caddyfile/compose take effect:
   ```bash
   docker compose -f docker-compose.prod.yml up -d --build caddy
   ```
   Then open **https://dietpi.YOUR-DOMAIN**. If the dashboard isn't installed
   yet, enable it on the Pi with `dietpi-software` (search "DietPi-Dashboard").

> **Verify the port.** `5252` is the DietPi-Dashboard default. If you changed it,
> override the upstream via the `DIETPI_UPSTREAM` env on the caddy service (e.g.
> `host.docker.internal:PORT`).

---

## Troubleshooting

- **502 / "no healthy origin"** — caddy or the app containers aren't up yet;
  check `docker compose ... ps` and the caddy/backend logs.
- **Discord login redirects to an error** — the redirect URI in the Discord
  portal must match `DISCORD_REDIRECT_URL` character-for-character (including
  `/api` and no trailing slash).
- **Logged out after every restart** — `SESSION_SECRET` is empty or changing;
  set a fixed random value.
- **Domain not resolving** — Cloudflare still shows the site as "Pending
  nameservers"; the simply.com nameserver change hasn't propagated yet.

---

## Hosting on the Raspberry Pi (DietPi)

Nothing above is Windows-specific, and the images are multi-arch (they build
natively on ARM64), so the move is mostly OS setup. Use a **64-bit** DietPi image
— required for the `arm64` container images and for Postgres to behave.

### One-time Pi setup

1. **Install Docker + Compose.** On DietPi, run `dietpi-software` and install
   **Docker** (ID 162) and **Docker Compose** (ID 134), or use the upstream
   script:
   ```bash
   curl -fsSL https://get.docker.com | sh
   sudo usermod -aG docker "$USER"      # then log out/in so the group applies
   ```
2. **Create the persistent deploy checkout.** This directory is what the deploy
   job updates on every push — it lives *outside* the runner's workspace and
   holds the gitignored `.env`. Cloning into your **home directory** keeps you as
   the owner with no `sudo chown` needed:
   ```bash
   git clone https://github.com/MickeyZacho/go-initiative-tracker.git ~/go-initiative-tracker
   ```
   > This must be the **same user** that runs the Actions runner (below), so the
   > runner can read/write it. Any path works as long as you point `DEPLOY_DIR`
   > at it in step 3 of the CD section.
3. **Recreate `.env`** in `~/go-initiative-tracker/.env` — same values as your
   desktop (the `TUNNEL_TOKEN` works from anywhere). This file is *not* in git,
   so it must be created on the Pi by hand and it persists across deploys.
4. **First manual launch** (proves the box works before automating):
   ```bash
   cd ~/go-initiative-tracker
   docker compose -f docker-compose.prod.yml up -d --build
   ```
   Once you can reach `https://YOUR-DOMAIN`, tear nothing down — the runner will
   just roll this same stack forward from here.

> **Low-RAM Pi (< ~2 GB):** the frontend (Vite) build is memory-hungry and the
> OOM killer may abort it. Add swap first: on DietPi run `dietpi-config` →
> *Advanced Options* → *Swapfile* and set ~2 GB (or `sudo fallocate -l 2G
> /swapfile && sudo chmod 600 /swapfile && sudo mkswap /swapfile && sudo swapon
> /swapfile`, then add it to `/etc/fstab`).

### Continuous deployment (self-hosted GitHub Actions runner)

The `deploy` job in [`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs
**on the Pi** through a self-hosted runner. It only fires after the backend and
frontend CI jobs pass, and only on a push to `main` (or a manual *Run workflow*).
Because the runner dials out to GitHub, this needs **no inbound ports** — it works
behind the Cloudflare Tunnel / CGNAT unchanged.

1. In GitHub: **repo → Settings → Actions → Runners → New self-hosted runner →
   Linux / ARM64**. GitHub shows a `curl` download command and a `./config.sh`
   command with a registration token — run them on the Pi as a **non-root** user
   (that user must be in the `docker` group from step 1):
   ```bash
   mkdir -p ~/actions-runner && cd ~/actions-runner
   # (paste the download + tar command GitHub gave you)
   ./config.sh --url https://github.com/MickeyZacho/go-initiative-tracker \
     --token <REG_TOKEN> --labels initiative-pi --name pi --unattended
   ```
   The **`initiative-pi` label is required** — the workflow targets
   `runs-on: [self-hosted, initiative-pi]`.
2. **Install it as a service** so it survives reboots and runs in the background:
   ```bash
   sudo ./svc.sh install "$USER"
   sudo ./svc.sh start
   ./svc.sh status          # should show "active (running)"
   ```
3. **Tell the deploy where the checkout is.** The job defaults to
   `/opt/initiative-tracker`, so if you cloned into your home directory (as in
   step 2 above) you **must** point it at that path: set a repository variable
   `DEPLOY_DIR` = `/home/<user>/go-initiative-tracker` (repo → Settings → Secrets
   and variables → Actions → **Variables** tab → *New repository variable*). It's
   a variable, not a secret — the path isn't sensitive. Skip this only if you
   used the exact default `/opt/initiative-tracker` path.

That's it. Push to `main` → CI runs on GitHub → the Pi pulls that exact commit,
rebuilds, restarts, and waits for the local smoke port (`127.0.0.1:8080`) to
answer before the job goes green. Watch it under the repo's **Actions** tab, or
tail it on the Pi:
```bash
cd ~/go-initiative-tracker
docker compose -f docker-compose.prod.yml logs -f backend
```

> **Security note.** A self-hosted runner executes whatever a workflow tells it
> to. Keep this repository **private**, or the runner could be abused by pull
> requests from forks. (The `deploy` job itself is guarded to run only on pushes
> to `main`, never on PRs — but a public repo still warrants a private runner.)
> Run the Pi on the desktop **or** the Pi against the same tunnel, not both at
> once; `docker compose ... down` on the desktop before letting the Pi take over.
