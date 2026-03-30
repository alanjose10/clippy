# Clippy — Design Spec

**Date:** 2026-03-30
**Status:** Approved

## Problem

Work machine is restricted from accessing external websites (Stack Overflow, AI tools, Reddit, etc.) but can reach other devices on the local network. Need a way to transfer code snippets, notes, and other text between the machines on the same network — with formatting preserved, syntax highlighted, and without any per-client installation requirements.

## Solution

A lightweight local-network clipboard service running on a Mac Mini via Docker Compose. Users access named "rooms" (e.g. `/work`, `/home`) from any browser. Snippets are stored in SQLite, encrypted at rest, and expire automatically. No accounts, no TLS complexity, no client-side installs beyond a browser.

---

## Stack

| Component | Choice | Reason |
|---|---|---|
| Language | Go | Single binary, easy Docker deployment |
| Database | SQLite via `modernc.org/sqlite` | Pure Go (no CGO), embedded, no external process |
| Frontend interactivity | HTMX | Minimal JS, all logic stays server-side in Go |
| Syntax highlighting | highlight.js | Bundled in binary, works offline |
| Static assets | `//go:embed` | All HTML templates, JS, CSS compiled into binary |
| Deployment | Docker Compose | Single service, mounted volume for DB persistence |

---

## Rooms

- A room is a named namespace identified by a short slug (e.g. `work`, `home`, `alan`). Slugs must be lowercase alphanumeric with hyphens only (`[a-z0-9-]+`), max 50 chars. Slugs starting with `_` are reserved for internal routes (e.g. `/_rooms`).
- Anyone who knows the room name can access it — no authentication in phase 1
- Rooms do not expire; only snippets within them expire
- **Phase 2 (future):** PIN-protected rooms. The `pin_hash` column is reserved in the schema now

---

## Snippets

Each snippet belongs to a room and has:
- **ID** — 6-char random alphanumeric (e.g. `x4kp9m`), used in the URL
- **Name** — optional user-set label (e.g. `db-migration`)
- **Content** — AES-256-GCM encrypted BLOB in the DB, decrypted on read
- **Language** — for syntax highlighting (e.g. `go`, `sql`, `js`)
- **Tags** — comma-separated string (e.g. `db,urgent`). Parsed to `[]string` in Go on read. Filtering is done in application code, not SQL.
- **Expires at** — per-snippet TTL. Default 24h, selectable at create time (1h / 6h / 24h / 7d)

---

## Database Schema

```sql
CREATE TABLE rooms (
  id         TEXT PRIMARY KEY,   -- slug: "work", "home"
  created_at DATETIME NOT NULL,
  pin_hash   TEXT                -- NULL in phase 1; bcrypt hash in phase 2
);

CREATE TABLE snippets (
  id          TEXT PRIMARY KEY,                         -- short random ID e.g. "x4kp"
  room_id     TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
  name        TEXT,                                     -- nullable user label
  content_enc BLOB NOT NULL,                            -- nonce || AES-256-GCM ciphertext
  language    TEXT NOT NULL DEFAULT 'plaintext',
  tags        TEXT NOT NULL DEFAULT '',                 -- comma-separated
  expires_at  DATETIME NOT NULL,
  created_at  DATETIME NOT NULL,
  updated_at  DATETIME NOT NULL
);

CREATE INDEX idx_snippets_room_expires ON snippets(room_id, expires_at);
```

---

## Routing

### User-navigable routes

```
GET    /{room}            List snippets in room (compact table + tag filter chips)
POST   /{room}            Create new snippet → redirect to /{room}/{id}
GET    /{room}/{id}       View snippet (read-only, syntax highlighted)
PUT    /{room}/{id}       Update snippet (name, content, language, tags, expiry)
GET    /{room}/{id}/raw   Plain text response (for curl / shell helper)
GET    /_rooms            Room management (list, create, delete)
POST   /_rooms            Create room → redirect to /{room}
DELETE /_rooms/{room}     Delete room and all its snippets
```

### HTMX-only fragment routes (not user-navigable)

```
GET    /{room}?tag=x&tag=y    Re-renders snippet table fragment (tag filtering)
GET    /{room}/_form          Returns new snippet form fragment (inline above table)
GET    /{room}/{id}/edit      Returns edit form fragment (swaps code block in view)
```

### Handler pattern

Every handler checks the `HX-Request` header:
- **Present** → render and return the relevant fragment template only
- **Absent** → render the full page template (layout + fragment embedded)

This means one handler serves both direct browser navigation and HTMX partial updates.

---

## UI

### Room view (`/{room}`)

Compact table layout. Columns: **Name / Preview**, **Language**, **Tags**, **Expires**.

- Tag filter chips rendered above the table from the unique tags collected during snippet iteration (no extra query)
- Multi-tag AND filtering: clicking multiple chips narrows results
- "+ New Snippet" button triggers `hx-get="/{room}/_form"` → inline form appears above table
- Expired snippets never appear (filtered in the SQL query by `expires_at > NOW()`)

### Snippet view (`/{room}/{id}`)

- Default: read-only, full syntax highlighting via highlight.js
- Header bar: breadcrumb (`work / db-migration`), language badge, **Edit**, **Copy**, **Raw** buttons
- Tag chips + expiry indicator below header
- "Edit" button: `hx-get="/{room}/{id}/edit"` → swaps code block for edit form with name, language selector, tag input, TTL selector, content textarea
- "Save": `hx-put="/{room}/{id}"` → server saves and returns updated view fragment

### Room management (`/_rooms`)

Simple page: list of all rooms with snippet counts, "+ Create Room" form (just a name input), delete button per room (with confirmation via HTMX `hx-confirm`).

---

## Encryption

- **Algorithm:** AES-256-GCM (authenticated encryption)
- **Key derivation:** `HKDF-SHA256(IKM=CLIPPY_SECRET, salt=nil, info="clippy-v1-snippet-content")` → 32-byte key, derived once at startup
- **Per-snippet:** Generate a random 12-byte nonce. Store `nonce || ciphertext` as the `content_enc` BLOB.
- **On read:** Split nonce from ciphertext, decrypt. If decryption fails, return HTTP 500 — never show corrupted content silently.
- **Key management:** `CLIPPY_SECRET` is set in `.env` alongside `docker-compose.yml`. If the secret is lost, all snippet content is unrecoverable. Document this prominently.

---

## Expiry

- A background goroutine runs on an hourly ticker
- Executes: `DELETE FROM snippets WHERE expires_at < datetime('now')`
- No soft-delete — expired rows are gone
- Snippets approaching expiry (< 1h) show the expiry time in red in the room table

---

## Configuration (env vars)

| Variable | Default | Description |
|---|---|---|
| `CLIPPY_SECRET` | required | Min 32 bytes. Used to derive encryption key. |
| `CLIPPY_PORT` | `8080` | HTTP listen port |
| `CLIPPY_DEFAULT_TTL_HOURS` | `24` | Default snippet TTL in hours |
| `CLIPPY_DB_PATH` | `/data/clippy.db` | Path to SQLite database file |

---

## Docker Compose

```yaml
services:
  clippy:
    image: clippy:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    env_file: .env
    restart: unless-stopped
```

The `./data` directory on the host persists the SQLite file across container restarts. The `.env` file holds `CLIPPY_SECRET` and any overrides.

---

## Project Structure

```
clippy/
├── cmd/clippy/         main.go — entry point, config loading, server start
├── internal/
│   ├── db/             SQLite setup, migrations, queries
│   ├── crypto/         AES-256-GCM encrypt/decrypt, HKDF key derivation
│   ├── handler/        HTTP handlers (rooms, snippets, fragments)
│   ├── model/          Room and Snippet types
│   └── worker/         Background expiry goroutine
├── templates/          Go HTML templates (full pages + HTMX fragments)
├── static/             htmx.min.js, highlight.js, styles.css
├── docker-compose.yml
├── Dockerfile
└── .env.example
```

All `templates/` and `static/` content is embedded into the binary at build time via `//go:embed`.

---

## Future Work (out of scope for phase 1)

- **PIN-protected rooms** — `pin_hash` column already reserved; add PIN entry UI and bcrypt verification
- **CLI tool** — Go binary that POSTs to `/{room}` and prints the snippet URL
- **Shell helper** — curl-based function requiring no installs:
  ```bash
  clip()    { curl -s -X POST http://mac-mini:8080/$1 --data-urlencode "content@-" ... }
  clipget() { curl -s http://mac-mini:8080/$1/$2/raw; }
  ```
- **Snippet pinning** — mark a snippet as never-expiring within a room

---

## Verification

1. `docker compose up` starts the service; `curl http://localhost:8080/_rooms` returns HTML
2. Create a room via `/_rooms`, navigate to `/{room}`, create a snippet
3. Verify snippet appears in the table with correct language badge and expiry
4. Open `/{room}/{id}/raw` — raw text only, no HTML
5. Edit a snippet via the Edit toggle, save, verify update persists
6. Verify tag filtering: create two snippets with different tags, filter by one tag — only matching snippet visible
7. Set a snippet TTL to 1h, advance system clock (or update DB directly), verify expiry worker removes it
8. Inspect SQLite DB directly — `content_enc` column should be binary, not plaintext
9. Stop container, restart — snippets persist from mounted volume
