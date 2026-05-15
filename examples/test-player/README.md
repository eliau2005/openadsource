# test-player

Single-file vanilla-JS verification harness for the `/vast` delivery endpoint.

## What it does

1. Fetches `GET <vast-base>/vast?ad_id=<uuid>` (defaults: `http://localhost:8088`, seeded internal_s3 ad UUID `00000000-0000-0000-0000-000000000003`).
2. Parses the response with `DOMParser`.
3. Pulls the first `<MediaFile>` URL and assigns it to a `<video>` element.
4. On the first `play` event, fires the `<Impression>` URL via `new Image()` so the impression hits `/track`.

## How to open it

Three options, in order of preference:

| | Command | URL |
|---|---|---|
| 1 | `docker compose --profile demo up -d test-player` | http://localhost:8090/test-player/ |
| 2 | `python -m http.server 8000` from this directory | http://localhost:8000/ |
| 3 | Double-click `index.html` | `file://.../index.html` |

Option 1 is recommended — it runs an nginx container on a dedicated host port and avoids any `file://` CORS quirks.

The adserver returns permissive `Access-Control-Allow-Origin: *` headers on `/vast` and `/track`, so all three options work.

## Query-string overrides

- `?ad_id=<uuid>` — load a different ad (e.g. the seeded `external_url` ad at `00000000-0000-0000-0000-000000000004`).
- `?vast=<base>` — point at a different adserver base URL (default `http://localhost:8088`).

## Known fragilities

The seeded sample MP4 default (`https://www.w3schools.com/html/mov_bbb.mp4`) is a public CC-BY clip. If it ever 404s, override with `SAMPLE_MP4_URL=...` in `.env` and re-run `docker compose --profile seed run --rm seed`.
