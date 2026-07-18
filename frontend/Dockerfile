# Frontend Dockerfile
FROM node:22-alpine AS builder
WORKDIR /app
COPY . .
RUN npm ci
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
