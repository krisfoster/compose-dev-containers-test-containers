# Quickstart Validation Guide: Nginx Front-Door Routing Layer

**Feature**: 014-nginx-front-door
**Prerequisite**: Docker Desktop running; repo cloned; `.env` file with `NGROK_AUTHTOKEN` (for
public-profile scenarios).

---

## Scenario 1 — Basic Stack Start (no ngrok)

```bash
docker compose up --build
```

**Expected**:
- nginx container starts and logs `start worker process`
- `http://localhost` (port 80) opens in a browser and shows the Crossy Whale landing page
- Landing page has three working buttons: "Play the game" → `/play`, "Host: show the QR code" → `/host`, "View the leaderboard" → `/leaderboard`
- No error logged by nginx or app during initial load

**Verify routing separation** (check nginx access logs):
```bash
docker compose logs nginx
```
- Requests for `/script.js`, `/*.glb`, etc. should be logged by nginx with no corresponding entry in `docker compose logs app`
- Requests for `/play`, `/host`, `/api/*` should appear in both nginx and app logs

---

## Scenario 2 — Game Play Path (Gate Check)

```bash
docker compose up --build
```

Open `http://localhost/play` in a browser that has no grant cookie.

**Expected**:
- Browser is redirected to `/host` (gate challenge — no grant cookie)
- After visiting `http://localhost/host`, a QR window is activated
- Revisit `http://localhost/play` — this time the game loads (grant cookie set during `/host` visit... actually this depends on whether `/host` grants access automatically. Adjust based on actual gate behaviour)

Actually the correct flow:
1. Open `http://localhost/host` — presenter activates QR window and gets a grant cookie
2. Open `http://localhost/play` — game loads with `window.__LEADERBOARD_TOKEN__` injected

**Verify token injection**:
```bash
# From browser devtools console (on /play page):
window.__LEADERBOARD_TOKEN__  # must NOT be a raw Go template placeholder
```

---

## Scenario 3 — Leaderboard Live Updates

```bash
docker compose up --build
```

Open `http://localhost/leaderboard` in a browser.

**Expected**:
- Leaderboard page loads
- "Standings" panel: React component renders (shows "No scores yet..." or existing scores)
- "Recent commits" panel: React component renders (shows commits or "no commits" message)
- Both panels update live without page reload when data changes

**Verify SSE connections via browser Network tab**:
- Two long-lived connections to `http://localhost/scores/stream` and `http://localhost/commits/stream`
- Status should remain open (not closed/errored) for at least 30 seconds

---

## Scenario 4 — Static Assets Served Without Go Involvement

```bash
docker compose up --build
# Clear app logs
docker compose logs app > /dev/null 2>&1
# Load the game
curl http://localhost/script.js -o /dev/null -w "%{http_code}\n"
curl http://localhost/three.module.js -o /dev/null -w "%{http_code}\n" 2>/dev/null || echo "(file may not exist, test with an actual game asset)"
```

Check app logs:
```bash
docker compose logs app | tail -20
```

**Expected**:
- Static game asset requests (JS bundles, GLB files, audio) return 200 from nginx
- `docker compose logs app` shows NO new entries for those requests — they were served by nginx directly
- `docker compose logs nginx` shows the requests were handled

---

## Scenario 5 — Score Submission End-to-End

```bash
docker compose up --build
```

1. Get a grant cookie (visit `http://localhost/host`)
2. Open `http://localhost/play` and play to score submission
3. Open `http://localhost/leaderboard` and confirm the new score appears in standings

Alternatively, submit a score directly:
```bash
curl -s -X POST http://localhost/api/leaderboard/scores \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-only-change-me" \
  -d '{"name":"test-player","score":999}' | jq .
```

**Expected**: `{"ok": true}` or similar acknowledgement; score appears in leaderboard within 5 seconds.

---

## Scenario 6 — ngrok Public Access (requires NGROK_AUTHTOKEN)

```bash
docker compose --profile public up --build
```

**Expected**:
- ngrok container establishes tunnel to `nginx:80` (not `app:8081`)
- Visit `http://localhost:4040` — ngrok inspection UI shows upstream as `nginx:80`
- `http://localhost/qr.png` returns a valid QR code image
- Scan QR code with a phone — opens `https://dockerdemo.ngrok.app/play?wid=...`
- Phone browser: gate check runs, grant obtained, game loads

**Verify correct upstream in ngrok**:
```bash
curl http://localhost:4040/api/tunnels | jq '.tunnels[0].config.addr'
# should output: "http://nginx:80"
```

---

## Scenario 7 — Leaderboard SSE Through ngrok (5-minute hold)

With `--profile public` active:
1. Open `https://dockerdemo.ngrok.app/leaderboard` in a browser
2. Wait 5 minutes
3. Confirm the standings and commits panels are still live (not showing disconnection or reload errors)

**Expected**: Both SSE connections survive 5 minutes; the `proxy_read_timeout 3600s` in nginx config allows the connection to remain open for a full booth session.

---

## Failure Scenarios

| Scenario | Expected behaviour |
|----------|--------------------|
| `scores-service` stopped mid-session | ScoresComponent falls back to polling (see `onerror` handler); no crash |
| `commits-service` stopped mid-session | CommitsComponent shows error/retry; no crash |
| `app` stopped mid-session | nginx returns 502 for dynamic routes; static game assets still served |
| nginx stopped mid-session | All access fails; expected (nginx is the single ingress) |

---

## Files Changed Checklist (verify after implementation)

- [ ] `nginx/Dockerfile` — exists; base image is `dhi.io/nginx:1-alpine3.24`
- [ ] `nginx/nginx.conf` — exists; routing rules match `contracts/routing-table.md`
- [ ] `docker-compose.yml` — nginx service present with `ports: ["80:80"]`
- [ ] `docker-compose.yml` — `SCORES_SERVICE_URL` default is `http://localhost` (not `http://localhost:8083`)
- [ ] `docker-compose.yml` — `COMMITS_SERVICE_URL` default is `http://localhost` (not `http://localhost:8082`)
- [ ] `ngrok.yml` — `addr: http://nginx:80` (not `http://app:8081`)
- [ ] `.specify/memory/constitution.md` — v1.4.0 with Routing Layer carve-out applied
