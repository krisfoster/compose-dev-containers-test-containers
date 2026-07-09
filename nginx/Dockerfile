# Docker Hardened Image for nginx (dhi.io/nginx:1-alpine3.24 matches the
# alpine3.24 base used by the rest of this project's runtime images).
#
# Build context: repo root — this Dockerfile is invoked with `context: .`
# in docker-compose.yml so it can COPY frontend/ and the gif alongside the
# nginx config.
#
# Static files copied into the image at build time (no runtime volume mount):
#   frontend/game/       → /usr/share/nginx/html/           (game assets)
#   container-obstacles.gif → /usr/share/nginx/html/       (gameplay demo)
#   frontend/leaderboard/ → /usr/share/nginx/html/leaderboard-assets/  (React bundles)
#
# The nginx.conf catch-all (try_files $uri =404) serves all of these paths
# as static files. Dynamic paths (/play, /host, /api/, /scores, /commits,
# /leaderboard, /) are proxied to the appropriate upstream service.

FROM dhi.io/nginx:1-alpine3.24
COPY frontend/game /usr/share/nginx/html
COPY container-obstacles.gif /usr/share/nginx/html/container-obstacles.gif
COPY frontend/leaderboard /usr/share/nginx/html/leaderboard-assets
COPY nginx/nginx.conf /etc/nginx/nginx.conf
