# Quickstart: QR-Gated Public Access to Crossy Whale

Validates the user stories in `spec.md` end-to-end against the running compose stack. Route
behavior referenced below is defined in `contracts/gate-http-contract.md`; state details are in
`data-model.md`.

## Prerequisites

- Docker Desktop (or equivalent) running.
- A free-tier ngrok authtoken in `.env` (`NGROK_AUTHTOKEN=...`), same as required by
  `001-host-webapp-ngrok`.
- A phone with a camera, on a network separate from the presenter's machine (e.g. cellular data),
  for the real scan-based checks.

## Start the stack

```bash
docker compose --profile public up
```

Expect: `redis` and `app` come up locally; `ngrok` establishes a public HTTPS URL pointed at the
app's gated listener. (Local-only smoke checks below also work with plain `docker compose up`,
i.e. without the `public` profile — the gate is irrelevant when there is no public endpoint.)

## Scenario 1 — Presenter activates and displays the QR code

1. Open `http://localhost:${WEB_PORT:-8080}/host` in a browser on the presenter's machine.
2. **Expect**: a QR code image is visible (first visit auto-activates a window per
   `contracts/gate-http-contract.md`'s `/host/rotate` semantics — no prior manual step needed).

## Scenario 2 — Attendee scans and reaches the game (User Story 1)

1. With Scenario 1's QR code displayed, scan it using the phone's native camera app (not a
   dedicated QR reader).
2. **Expect**: the phone's browser opens directly to the game, playable immediately, within the
   10-second budget in SC-001.
3. Play a round to completion, then start a second round without leaving the page.
4. **Expect**: no re-scan or re-prompt is required for the second round (FR-005).

## Scenario 3 — Public endpoint blocks an un-scanned visitor (User Story 2)

1. On a *different* device/browser (or the same phone in a private/incognito tab, to guarantee no
   grant cookie is present), open the bare public URL ngrok reports (no `?w=...`).
2. **Expect**: a "scan the QR code to play" message, `403` — not the game, not a generic error
   page (SC-002).
3. On the presenter's machine, confirm `http://localhost:${WEB_PORT:-8080}/play` still loads the
   game directly with no gate involved (FR-004).

## Scenario 4 — Rotation cuts off the old code (User Story 3)

1. Note the current QR code's encoded URL (e.g. by decoding the PNG or copying the link).
2. From `/host`, trigger rotation.
3. Attempt to open the *previous* URL from Scenario 4 step 1 in a fresh/incognito browser tab.
4. **Expect**: blocked, same as Scenario 3 (SC-003 — effectively immediate).
5. Scan the *newly displayed* QR code from a fresh device/tab.
6. **Expect**: access granted normally, as in Scenario 2.
7. Return to the device from Scenario 2 (already holding a grant from before rotation) and keep
   playing.
8. **Expect**: still works uninterrupted (FR-008).

## Scenario 5 — Concurrent visitors get independent, identifiable grants (SC-006)

1. Repeat Scenario 2 from two different devices in quick succession, using the same active QR
   code.
2. **Expect**: both reach the game independently; neither is blocked by the other.
3. Inspect each device's `cw_grant` cookie value (e.g. via browser devtools).
4. **Expect**: the two decode to different `grant_id` values, with no collision.

## Scenario 6 — Automatic expiry with no presenter action (SC-004)

1. Activate a window, then wait past its validity period (or temporarily lower the window TTL via
   its environment variable for a faster check) without visiting `/host` again.
2. Attempt to use that window's QR code from a fresh/incognito tab.
3. **Expect**: blocked, identical to Scenario 3 (FR-006, FR-009) — no presenter action was taken.

## Automated coverage

Go tests cover the pieces above that don't require an actual camera or a second physical device:
window activation/rotation/expiry logic against a real Redis via Testcontainers-go (constitution
Principle III), and grant cookie signing/verification as pure unit tests. Scenarios 2 and 5's
"scan with a real phone camera" step is inherently manual and is the one piece this quickstart
covers that a test suite cannot (constitution Principle IV).
