FROM docker.io/library/node:24-alpine AS builder

WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM docker.io/nginxinc/nginx-unprivileged:1.29-alpine

COPY --chmod=644 web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8080
