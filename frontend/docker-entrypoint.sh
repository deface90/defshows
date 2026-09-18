#!/bin/sh
# Render the runtime config from the template, substituting only known vars.
# Runs via the official nginx image's /docker-entrypoint.d/ hook before nginx starts.
set -e

: "${API_BASE_URL:=http://localhost:8080}"
export API_BASE_URL

envsubst '${API_BASE_URL}' \
  < /usr/share/nginx/html/env.template.js \
  > /usr/share/nginx/html/env.js

echo "defshows: rendered env.js with API_BASE_URL=${API_BASE_URL}"
