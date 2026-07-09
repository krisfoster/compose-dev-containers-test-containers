# Contract: `/auth/check` — Grant Validation Endpoint

**Feature**: 016-nginx-auth-score-integrity  
**Date**: 2026-07-09  
**Consumer**: nginx `auth_request` sub-request  
**Producer**: Go app (`app/main.go`)

---

## Overview

`GET /auth/check` is an **internal-only** HTTP endpoint. External clients cannot call it directly
(nginx marks it `internal;`). It exists solely as the target of nginx's `auth_request` directive,
which calls it as a synchronous sub-request before forwarding requests to `/api/leaderboard/scores`.

The endpoint validates the `cw_grant` session cookie and returns a binary result that nginx uses
to allow or deny the original request.

---

## Request

```
GET /auth/check HTTP/1.1
Cookie: cw_grant=<signed-grant-value>
```

**Method**: `GET` only. Other methods return `405 Method Not Allowed`.

**Headers forwarded by nginx**:
- `Cookie: $http_cookie` — the original request's full cookie header, forwarded via
  `proxy_set_header Cookie $http_cookie;` in the `auth_request` target location.

**Request body**: None. nginx sets `proxy_pass_request_body off;` for the sub-request location.

**Authentication**: None. This endpoint is only reachable as a nginx internal sub-request; it is
not itself authenticated.

---

## Responses

### 200 OK — Grant Valid

The `cw_grant` cookie is present, HMAC-valid, and within its configured lifetime.

```
HTTP/1.1 200 OK
Content-Length: 0
```

nginx interprets this as authorisation success and forwards the original request to the upstream.

### 401 Unauthorized — Grant Invalid or Absent

Any of: cookie absent; cookie value does not parse; HMAC check fails; grant age exceeds
`GRANT_LIFETIME`.

```
HTTP/1.1 401 Unauthorized
Content-Length: 0
```

nginx interprets this as authorisation failure and returns `401` to the original client. The Go
app's score submission handler is never reached.

### 405 Method Not Allowed

```
HTTP/1.1 405 Method Not Allowed
Allow: GET
Content-Length: 0
```

### (Unreachable) 403 Forbidden

nginx's `auth_request` also treats a `403` response as access denied. This response code is not
produced by the endpoint but is listed here for completeness because nginx docs mention it.

---

## nginx Configuration (reference)

```nginx
location /api/leaderboard/scores {
    auth_request /auth/check;
    proxy_pass http://app:8080;
    proxy_set_header Host $host;
    proxy_set_header Cookie $http_cookie;
}

location = /auth/check {
    internal;
    proxy_pass http://app:8080/auth/check;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header Cookie $http_cookie;
}
```

---

## Validation Rules

The endpoint applies the same grant validation as `gate.Signer.Verify()`:

1. The `cw_grant` cookie must be present in the request.
2. The cookie value must have the form `<base64url-payload>.<base64url-mac>`.
3. The HMAC-SHA256 of the encoded payload using `GRANT_COOKIE_SECRET` must equal the decoded MAC.
4. The `issued_at` field in the decoded payload must be within `GRANT_LIFETIME` of the current
   server time.

All four conditions must hold for a 200 response. Any failure produces 401.

---

## Security Properties

- **No cookie issuance**: The endpoint is read-only. It never sets, modifies, or deletes cookies.
- **No state mutation**: No Redis writes occur. No grant is refreshed or extended.
- **Timing**: The HMAC comparison uses `hmac.Equal` (constant-time) from the existing `gate`
  package, preventing timing-based secret inference.
- **Internal only**: The `internal;` nginx directive blocks direct external requests to
  `/auth/check`. A client cannot call it directly to test cookie values.
- **Failure closed**: An absent or malformed secret (e.g., `GRANT_COOKIE_SECRET` not set)
  causes `gate.Signer.Verify()` to reject all cookies, not accept them.

---

## Relationship to Other Contracts

- Replaces the `X-Leaderboard-Token` credential check described in
  `specs/003-leaderboard-score-submission/contracts/` (that contract is now retired).
- The `cw_grant` cookie format is defined in
  `specs/002-qr-gated-access/contracts/gate-http-contract.md`.
- The score submission endpoint (`POST /api/leaderboard/scores`) contract in
  `specs/004-leaderboard-page/contracts/leaderboard-openapi.yaml` is updated by this feature:
  the `X-Leaderboard-Token` header is no longer required or accepted.
