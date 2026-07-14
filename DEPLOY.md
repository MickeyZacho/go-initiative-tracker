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
| `Caddyfile` | Reverse proxy: serves the frontend, strips `/api` → backend |
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

## Later: moving to a Raspberry Pi

Nothing above is Windows-specific, and the images are multi-arch (they build
natively on ARM), so the move is mostly OS setup.

**Recommended OS:** **Raspberry Pi OS Lite (64-bit)** — the mainstream, best-
supported choice, headless (no desktop). If you want something even leaner,
**DietPi** (64-bit) is a good alternative with a smaller footprint. Use 64-bit
either way (needed for the `arm64` container images and for Postgres to behave).

1. Flash Raspberry Pi OS Lite (64-bit) with **Raspberry Pi Imager** (it lets you
   preset the hostname, SSH, and Wi-Fi before first boot — handy since you're
   headless).
2. SSH in and install Docker:
   ```bash
   curl -fsSL https://get.docker.com | sh
   sudo usermod -aG docker $USER   # log out/in afterwards
   ```
3. Copy the project to the Pi (`git clone` your repo, or `scp` the folder).
4. Recreate `.env` on the Pi (same values as your desktop — the `TUNNEL_TOKEN`
   works from anywhere).
5. Run the same command:
   ```bash
   docker compose -f docker-compose.prod.yml up -d --build
   ```

Run it on the Pi **or** the desktop — not both against the same tunnel at once.
When you're ready to switch, `docker compose ... down` on the desktop first.

> On a Pi with under ~2 GB RAM the frontend (Vite) build can be memory-hungry.
> If the build gets killed, either add swap, or build the images on your desktop
> and push them to a registry / load them onto the Pi instead of building there.
