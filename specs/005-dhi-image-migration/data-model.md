# Phase 1 Data Model: DHI Migration

This feature introduces no runtime data model (no new entities, storage, or schema in the app).
The only "data" is the **Image Inventory** — a documentation artifact (not code) that records the
migration status of every container image. Its shape is defined here and instantiated in
[`contracts/image-inventory.md`](./contracts/image-inventory.md).

## Entity: Image Inventory Entry

One record per container image the project references anywhere (Dockerfile stages, compose services,
test code, docs).

| Field | Type | Description | Rules |
|-------|------|-------------|-------|
| `role` | string | Logical role of the image | One of: `app-build`, `app-runtime`, `redis`, `tunnel`. Unique per entry. |
| `previous_ref` | string | Image reference before migration | e.g. `golang:1.25-alpine`. |
| `status` | enum | Migration outcome | `migrated` \| `exempt`. |
| `new_ref` | string \| null | DHI reference after migration | Required (`dhi.io/...`) when `status = migrated`; `null` when `exempt`. |
| `version_line` | string | Major/minor line in use | e.g. `1.25`, `8.x`, `3` (ngrok). |
| `used_in` | list of paths | Where the image is referenced | e.g. `app/Dockerfile`, `docker-compose.yml`, `store_test.go`. |
| `on_core_demo_path` | boolean | Whether it runs in the default (non-`public`) demo | ngrok is `false`; app + redis are `true`. |
| `rationale` | string | Reason for the status | Required for every `exempt` entry; short note for `migrated`. |

### Validation rules (from spec requirements)

- **Completeness (FR-008, SC-002)**: Every image found by a repo-wide search for image references
  MUST have exactly one entry. Cross-check: `grep`-able references reconcile 1:1 with the inventory.
- **Migrated coverage (FR-001, SC-001)**: Every entry whose image has a DHI equivalent MUST have
  `status = migrated` with a `dhi.io/...` `new_ref`.
- **Exemption honesty (FR-009, SC-006)**: Every `exempt` entry MUST state a `rationale`. Exactly one
  entry (`tunnel`/ngrok) is expected to be `exempt` for lack of a DHI equivalent.
- **Version parity (FR-010)**: `version_line` changes vs `previous_ref` (Redis `7`→`8`) MUST be
  called out in `rationale`.

### State (this migration's expected instance)

| role | status | version_line | on_core_demo_path |
|------|--------|--------------|-------------------|
| app-build | migrated | 1.25 | true (build-time) |
| app-runtime | migrated | static (date-tag) | true |
| redis | migrated | 8.x (was 7.x) | true |
| tunnel (ngrok) | exempt | 3 | false |
