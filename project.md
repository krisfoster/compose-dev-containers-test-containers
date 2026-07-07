# Vibe Room

A shared-screen demo app. Participants scan a QR code on the wall, type a short "vibe" phrase into their phone, and a local LLM turns it into a color, emoji, and energy score. The wall renders every vibe as an animated blob and the aggregate mood shifts in real time.

Runs entirely on a laptop. Public access is via ngrok. If no one can connect, the laptop itself doubles as an input surface so the presenter can still drive the demo.

## Goals

- Visible: one big animated canvas that people want to point at.
- Fun: instant feedback loop, sub-2s from tap to blob on the wall.
- AI-flavored: a real local model classifies every input, no rules engine.
- Self-contained: `docker compose up` is the only setup command.
- Demo-resilient: works when the venue wifi blocks client-to-client, when ngrok is throttled, when there's no internet at all.

## Runtime modes

The same binary supports three modes, selected by `APP_MODE`:

| Mode     | Access path                                        | When to use                                                                 |
|----------|----------------------------------------------------|-----------------------------------------------------------------------------|
| `public` | Public ngrok URL shown as QR on the wall           | Default. Attendees join from their phones over the internet.                |
| `lan`    | Laptop LAN IP shown as QR on the wall              | Venue wifi allows client-to-client. Avoids the ngrok interstitial.          |
| `kiosk`  | No QR. The wall shows the visual plus an input bar | Fallback when nothing external works. Presenter types on the laptop itself. |

Mode is chosen at startup. The wall page renders differently depending on mode, but the backend behavior is identical: every submission funnels through the same pipeline.

### Kiosk fallback in detail

Even in `public` or `lan` mode, the laptop always exposes a local-only `/host` route protected by a short token. The presenter can open it in a second browser tab and submit vibes on behalf of the audience ("via host"). This is what lets the demo survive when nobody in the room can reach the app: the presenter walks around, asks people for their vibe phrase, types it in, and the wall behind them still lights up. Same UX, one degree of separation.

## High-level architecture

```
                                     +-------------------+
                                     |  Wall (browser)   |
                                     |  fullscreen canvas |
                                     +---------+---------+
                                               ^
                                               | WebSocket (vibes stream)
                                               |
   Phones            +------------------+      |      +----------------+
  (join page) --->   |   ngrok tunnel   | ---> |      |   Go backend   |
                     +------------------+      +----->|                |
   Host laptop  ---> local :8080 (kiosk/host) ------->|  /vibe /ws /mood
  (host page)                                          |  /qr  /host
                                                       +--------+-------+
                                                                |
                                                +---------------+---------------+
                                                |                               |
                                        +-------v--------+             +--------v-------+
                                        |     Redis      |             |     Ollama     |
                                        | streams, sorted|             |  local LLM     |
                                        | sets, hashes,  |             |  llama3.2:3b   |
                                        | pubsub, HLL    |             |  (or similar)  |
                                        +----------------+             +----------------+
```

Data flow per submission:

1. Phone (or host page) POSTs `{text, handle}` to `/vibe`.
2. Go server calls Ollama with a tight prompt asking for JSON.
3. Response parsed into `{emoji, hex, energy, warmth, label}`.
4. Written to Redis: stream (audit log), hash (active vibe, TTL 90s), sorted set (mood window), pubsub (fan-out), HyperLogLog (unique contributors).
5. Every open Wall WebSocket receives the new vibe event.
6. Wall canvas spawns a blob, biases global color toward the aggregate.

## Components to build

Grouped by concern. Each is a discrete buildable unit.

### 1. Go backend (`cmd/viberoom`, `internal/`)

- **HTTP server**: routes `/vibe`, `/mood`, `/ws`, `/host`, `/qr.png`, `/public-url`, static files.
- **WebSocket hub**: manages Wall connections, fans out pubsub events.
- **Vibe classifier**: prompt template, Ollama client, JSON parse, one retry on malformed output, fallback color if the model refuses.
- **Redis adapter**: thin wrappers over stream XADD, sorted-set ZADD/ZRANGEBYSCORE, hash HSET with TTL, PUBLISH/SUBSCRIBE, PFADD/PFCOUNT.
- **Mood aggregator**: reads the last N seconds from the sorted set, returns average warmth/energy plus top labels.
- **Tunnel probe**: on startup in `public` mode, polls the ngrok agent API (`http://ngrok:4040/api/tunnels`) until a public URL appears, caches it in Redis, and exposes it at `/public-url`.
- **LAN IP probe**: in `lan` mode, resolves the laptop's LAN IP for the QR code.
- **Host auth**: `/host` requires an `X-Host-Token` header or cookie set from a token printed to the server log on startup.

### 2. Frontend (`web/static/`, plain HTML + vanilla JS)

- **`wall.html`**: fullscreen canvas. Connects to `/ws`, renders blobs, drives the background color toward the aggregate. Shows the QR code (fetched from `/qr.png`) in a corner in `public` and `lan` modes. In `kiosk` mode, shows an input bar instead.
- **`join.html`**: minimal mobile page, single text input, submits to `/vibe`, shows a short confirmation animation.
- **`host.html`**: same as join but with the host token attached and a small "posted as host" indicator.
- **Canvas renderer**: pick one, PixiJS or vanilla Canvas 2D. PixiJS if we want particle effects, plain Canvas if we want the codebase small.
- **QR image**: server-rendered PNG so the wall just does `<img src="/qr.png">`.

### 3. Local LLM (Ollama)

- Model choice: start with `llama3.2:3b` for speed on a laptop, upgrade to `qwen2.5:7b` if quality matters more than latency.
- Prompt returns strict JSON. Temperature low. Max tokens capped small.
- Warm-up ping on server boot so the first user submission isn't slow.

### 4. Redis

- Single instance, no persistence needed for a demo. If we want the "time travel slider" later, enable AOF.
- Keys:
  - `stream:vibes` (Redis Stream, append-only log)
  - `vibe:<id>` (Hash, TTL 90s, live blob state)
  - `mood:window` (Sorted Set, score = unix timestamp, member = vibe id)
  - `contributors:daily` (HyperLogLog, unique handles today)
  - `channel:vibes` (Pub/Sub for WS fan-out)
  - `public-url` (String, current tunnel URL)

### 5. Tunnel (ngrok)

- Runs as a compose service. Config file mounts the auth token from env.
- Forwards the app's `:8080` to a public HTTPS URL.
- The Go backend reads the URL from the ngrok agent API rather than parsing logs.
- If the tunnel is down, `/public-url` returns 503 and the wall falls back to a "Ask the host to submit for you" banner. Everything else keeps working.

## Repo layout

```
vibe-room/
├── README.md
├── project.md                  <-- this file
├── docker-compose.yml
├── .env.example
├── Dockerfile                  (Go multi-stage build)
├── go.mod / go.sum
├── cmd/
│   └── viberoom/
│       └── main.go             (entry point, flag/env parse, wire dependencies)
├── internal/
│   ├── server/                 (HTTP handlers, WS hub, middleware)
│   ├── vibe/                   (LLM prompt, JSON schema, classifier)
│   ├── ollama/                 (thin HTTP client for local model)
│   ├── store/                  (Redis wrappers)
│   ├── mood/                   (aggregation)
│   ├── tunnel/                 (ngrok agent API probe)
│   └── qr/                     (QR code PNG rendering)
├── web/
│   └── static/
│       ├── wall.html
│       ├── join.html
│       ├── host.html
│       ├── wall.js             (canvas + WS client)
│       ├── join.js
│       ├── host.js
│       └── styles.css
└── scripts/
    ├── ollama-pull.sh          (helper to pre-pull the model image)
    └── qr-preview.sh           (dumps the current QR to a terminal)
```

## Compose services

| Service | Image                     | Purpose                                    | Depends on       | Ports (host) |
|---------|---------------------------|--------------------------------------------|------------------|--------------|
| redis   | `dhi.io/redis:8-alpine`   | State, pubsub, stream                      |                  | 6379         |
| ollama  | `ollama/ollama:latest`    | Local LLM inference                        |                  | 11434        |
| app     | built from local Dockerfile (DHI: `dhi.io/golang:1.25-alpine-dev` → `dhi.io/static:20260611-alpine3.24`) | Go backend, static files, WS hub | redis, ollama | 8080 |
| ngrok   | `ngrok/ngrok:3` (exempt)  | Public tunnel to `app:8080`                | app              | 4040 (agent) |

Notes:

- All container images with a hardened equivalent are sourced from **Docker Hardened Images (DHI)**, pulled free from `dhi.io` after `docker login dhi.io`. See the full migration status in [`specs/005-dhi-image-migration/contracts/image-inventory.md`](specs/005-dhi-image-migration/contracts/image-inventory.md).
- `ngrok` has no DHI equivalent and is **exempt** — it runs only in `public` mode (via `--profile public`) and is off the core local demo path.
- Redis moved from the public `redis:7-alpine` to `dhi.io/redis:8-alpine` (a documented 7→8 bump: no hardened Redis 7.x exists; hardened Redis starts at 8.x).
- `ollama` benefits from a mounted volume so the model doesn't re-download every rebuild.
- The `app` container reads the ngrok URL by hitting `http://ngrok:4040/api/tunnels` on the internal compose network.
- No host network mode required. All internal traffic uses the compose network.

## Environment variables (`.env.example`)

| Name                | Default          | Notes                                                     |
|---------------------|------------------|-----------------------------------------------------------|
| `APP_MODE`          | `public`         | `public`, `lan`, or `kiosk`                               |
| `APP_PORT`          | `8080`           |                                                            |
| `REDIS_URL`         | `redis://redis:6379/0` |                                                     |
| `OLLAMA_URL`        | `http://ollama:11434` |                                                      |
| `OLLAMA_MODEL`      | `llama3.2:3b`    |                                                            |
| `NGROK_AUTHTOKEN`   | (unset)          | Required for `public` mode                                |
| `NGROK_DOMAIN`      | (unset)          | Optional. Reserved domain if you have a paid ngrok plan   |
| `HOST_TOKEN`        | random on boot   | If unset, generated and printed to the server log         |
| `VIBE_TTL_SECONDS`  | `90`             | How long each blob lives on the wall                      |
| `MOOD_WINDOW_SECONDS` | `120`          | Sliding window used to compute aggregate mood             |

## Build checklist

Rough order, each item is one focused unit of work:

1. Compose file plus `.env.example`, Dockerfile skeleton, `docker compose up` boots empty containers.
2. Redis wired, Go server exposes `/healthz` that pings Redis.
3. Ollama wired, `/healthz` also confirms the model is loaded.
4. `POST /vibe` accepts JSON, writes raw text to stream, returns 202. No AI yet.
5. Vibe classifier: LLM call, JSON parse, retry, fallback. Now `/vibe` returns the classified vibe.
6. WebSocket hub, Redis pubsub bridge, `wall.html` connects and logs events to console.
7. Canvas renderer on the wall page. Blobs spawn, fade, colors shift.
8. Mood aggregator plus `/mood`, wall uses it to bias background color.
9. `join.html` mobile page, minimal form, submits to `/vibe`.
10. QR code endpoint plus wall corner display.
11. ngrok compose service, tunnel probe in Go, `/public-url`, QR uses it.
12. `APP_MODE=lan` support, LAN IP detection.
13. `APP_MODE=kiosk` support, `host.html`, host token flow, wall inline input bar.
14. Polish: HyperLogLog contributor count, top labels ticker, "vibe of the hour" freeze frame.

Items 1 to 9 are the demoable core. 10 and 11 unlock the "phones scanning QR" experience. 12 and 13 are the demo-resilience layer.

## Open decisions to make before building

- **Canvas library**: PixiJS (nicer particles, larger footprint) vs vanilla Canvas 2D (smaller, more manual).
- **Model**: `llama3.2:3b` (fast, sometimes messy JSON) vs `qwen2.5:7b` (slower, cleaner output). Could support both via env.
- **JSON schema enforcement**: rely on prompting alone, or use Ollama's structured output / grammar mode.
- **Host token UX**: bookmark with token in URL, or a manual entry screen. Bookmark is faster on demo day.
- **Persistence**: ephemeral by default. If we want the "time travel" feature, enable Redis AOF and a snapshot job.
- **Analytics**: log every submission to a local JSONL file for post-demo review? Cheap to add.

## What "done" looks like for a demo

- `docker compose --profile public up` starts everything.
- Within 10 seconds the wall shows an animated background, a QR code, and a live "0 vibes" counter.
- Scanning the QR opens a mobile page. Submitting a phrase makes a blob appear on the wall within 2 seconds.
- The presenter has `http://localhost:8080/host?token=...` bookmarked as a fallback and can submit from the laptop directly.
- Pulling the network cable does not crash the app. The QR turns into a "host is submitting for you" banner and the wall keeps animating whatever the host types.
