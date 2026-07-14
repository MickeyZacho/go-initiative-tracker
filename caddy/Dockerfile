# Bakes the reverse-proxy config into the image so the running prod stack has no
# host-file dependency on the Caddyfile (survives moving/deleting the project
# directory and reboots).
FROM caddy:2-alpine
COPY Caddyfile /etc/caddy/Caddyfile
