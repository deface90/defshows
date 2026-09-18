# Multi-stage build for the defShows SPA.
# Build with: docker build -f Dockerfile . (context is the frontend/ directory)
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
# envsubst (gettext) renders env.js from the template at container start.
RUN apk add --no-cache gettext
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
# The official nginx entrypoint runs /docker-entrypoint.d/*.sh before starting nginx.
COPY docker-entrypoint.sh /docker-entrypoint.d/40-defshows-env.sh
RUN chmod +x /docker-entrypoint.d/40-defshows-env.sh
EXPOSE 80
