# Whale Runner (working title)

A browser-based Crossy Road clone. The Docker whale is the player. Attendees scan a QR code on a wall, join under a display name, play in their browser, and their score lands on a live leaderboard powered by a Go and Redis backend. When the network is patchy the game keeps working; when the demo laptop has no internet at all, the presenter can still play and demo on the laptop directly.

Everything runs on the presenter's laptop under `docker compose`. Public URLs come from ngrok when needed. Access is gated so only people who can see the QR code (or reach the laptop) can play.

## Player flow

1. Wall or projector displays a QR code and the current leaderboard.
2. Player scans, lands on a join page.
3. Enters a display name and picks (or is assigned) a disambiguator emoji so "kris turtle" and "kris fox" don't collide.
4. Game screen loads. The whale hops across roads, rivers, trains, dodging traffic and logs.
5. Player dies. Score is sent to the API. Leaderboard on the wall updates within a second.
6. Player sees their placement and a "play again" button.

## High-level architecture

- **Game client (browser)**: static single-page app. Renders the game. Owns all game logic and physics client-side. Talks to the Go API only for join, score submit, and leaderboard fetch. Opens a WebSocket for live leaderboard updates.
- **Wall page**: separate route on the same server. Full-screen QR plus live leaderboard. Meant for the projector.
- **Host page**: local-only page for the presenter. Buttons for rotating the QR, clearing the leaderboard, and pausing new joins.
- **Go server**: serves the SPA, wall page, host page, and JSON API. Runs the WebSocket hub. Handles cookie signing and access-window checks.
- **Redis**: sessions, active QR window, leaderboard sorted set, pubsub channel for live updates.
- **ngrok**: exposes the Go server as a public HTTPS URL. Only started in `public` mode via a compose profile.

```
   Phones  -->  [ ngrok ]  -->
                              \
   Laptop (host/kiosk)  ---->  [  Go server :8080  ]
                              /         |
                                        |
                             +----------+----------+
                             |                     |
                         +---v---+          +------v------+
                         | Redis |          | static SPA  |
                         +-------+          +-------------+
```

## Access control (only visible-QR people can play)

Three layers, stacked:

1. **QR-window tokens**. The QR encodes a URL like `https://<ngrok>/play?w=<window_id>`. `window_id` is a short random string, valid for a bounded time (default 15 minutes). Scans within the window are accepted. Stale window IDs return a friendly "ask the host for a new QR" screen.
2. **Presenter-triggered rotation**. The host page has a "rotate QR" button. Rotation invalidates the previous `window_id` immediately, so a screenshot of the QR taken 20 minutes ago cannot be used after the presenter refreshes.
3. **Signed session cookies**. Once a player joins successfully, the server sets a short-lived signed cookie. Subsequent API calls (score submit, leaderboard read from the game) rely on the cookie, not the token in the URL. Copy-pasting the URL to a friend after the fact does not grant them a persistent session; they still need to hit the join page inside the current window.

This is not bulletproof against a motivated attacker, but the layered flow means URL screenshots go stale on their own, the presenter can nuke access with one click, and sharing the URL alone does not carry a session forward.

The wall page is served on a separate route and is gated to either localhost or a wall-specific token, so random ngrok visitors cannot scrape the leaderboard by guessing URLs.

## Patchy network and offline behavior

The game runs entirely in the browser once the SPA is loaded. Everything else is best-effort:

- **Score submit fails**: client retries with backoff. If it still fails, the score is stored in localStorage and flushed on the next successful API call.
- **Leaderboard fetch fails**: the last known leaderboard is shown with a small "stale" badge.
- **WebSocket drops**: auto-reconnect with backoff. On reconnect the client resyncs the leaderboard once.
- **Whole internet drops mid-demo**: the presenter switches the QR from the ngrok URL to the laptop's LAN IP, or to a kiosk-mode QR that just points to `http://localhost:8080/play`. Same Go server, no code change.
- **Laptop with no wifi at all**: the presenter plays on the laptop directly. The Go server and Redis are both local, so the story still holds.

The game itself has no server dependency for movement or physics. The server only owns identity (who joined) and outcome (final score).

## Components to build

- **Game client**: game engine setup, voxel-style whale model, lane generator, traffic and log patterns, controls (arrow keys on desktop, tap zones on mobile), scoring, death handling, HUD.
- **Whale model**: chunky voxel version of the Docker logo. Either a hand-authored .glb or a small procedural voxel array.
- **Join page**: name input, emoji picker or auto-assigned emoji, submits to `/api/join`, transitions into the game.
- **Wall page**: full-screen leaderboard, QR code, "now playing" indicator.
- **Host page**: local-only. Rotate QR, clear leaderboard, pause joins, view active sessions.
- **Go server**: HTTP routes, WebSocket hub, Redis client, cookie signing, static asset serving, ngrok URL probe.
- **Redis schema and helpers**: sessions, windows, leaderboard, pubsub.
- **Compose setup**: `redis`, `app`, and `ngrok` (behind a `public` profile).
- **QR renderer**: server-side PNG so the wall just does `<img src="/qr.png">`.

## API surface (first pass)

- `POST /api/join` -> body `{name, emoji, window_id}`. Sets signed session cookie on success. 403 if the window is stale.
- `POST /api/score` -> body `{score}`. Auth: session cookie. Writes to leaderboard sorted set, publishes an update.
- `GET /api/leaderboard` -> top N entries.
- `GET /ws` -> subscribes to live leaderboard updates.
- `GET /play` -> the game SPA.
- `GET /wall` -> the projector page.
- `GET /host/*` -> local-only presenter routes.
- `GET /qr.png` -> current QR image.

## Redis data model (first pass)

- `session:<id>` (Hash): name, emoji, joined_at.
- `window:current` (String): current active `window_id`.
- `window:<id>` (String, TTL): existence proves validity.
- `leaderboard:current` (Sorted Set): member = session_id, score = points.
- `channel:leaderboard` (Pub/Sub): fan-out for live updates.

## Compose services

- `redis` — data and pubsub.
- `app` — Go server, serves API and static SPA.
- `ngrok` — behind the `public` profile, only started when a public URL is wanted.

No local LLM in this design. The original brief called for an AI element; the natural hook here is commentary (a small local model narrates the leaderboard: "kris-turtle just knocked mo-fox off the podium"). It can be added later as its own service without changing the core, so I've left it out for now.

## Open decisions

- **Game engine**: three.js (voxel 3D, closest to real Crossy Road) vs Phaser (2D, faster to build) vs pure Canvas 2D (smallest footprint).
- **Whale model**: hand-authored voxel .glb vs procedural voxel array baked in code.
- **Wall page access**: localhost-only, or same window-token as players?
- **Session persistence**: expire on game end, or persist so a player can rejoin under the same name across attempts?
- **Leaderboard scope**: one global board that stays until reset, or resettable per event via the host page?
- **Score integrity**: trust the client, or add a lightweight play-time sanity check (min game duration, max score-per-second)?
- **AI hook**: skip entirely, or wire in a commentary service later?

## What "done" looks like for a demo

- `docker compose --profile public up` starts everything.
- Presenter opens the wall page on the projector. QR and empty leaderboard visible within 10 seconds.
- Attendee scans, enters name plus emoji, plays, dies. Score appears on the leaderboard within a second.
- Rotating the QR from the host page immediately kills further joins on the old URL.
- Killing wifi mid-demo: presenter switches to the laptop and plays locally. Leaderboard and story stay intact.
