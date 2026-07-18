# syntax=docker/dockerfile:1
# Frontend Dockerfile
FROM node:22-alpine AS builder
WORKDIR /app
# Install deps first, keyed only on the lockfiles. A source-only change then
# reuses this layer instead of triggering a full `npm ci` reinstall on every
# commit. The cache mount persists npm's download cache across builds.
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
# Now the source. Changes here reuse the cached node_modules layer above.
COPY . .
# Stamp the bundle with the git short SHA (baked in via Vite's `define`).
# Passed in from docker-compose / CI; defaults to "dev" for a plain build.
ARG GIT_SHA=dev
ENV GIT_SHA=${GIT_SHA}
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
# SPA fallback so deep links / reloads on client-side routes serve index.html
# instead of 404ing.
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
