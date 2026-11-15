# GoMusic API

GoMusic reads public playlist metadata. It exports text; transferring tracks to another platform is handled by the user's chosen transfer tool. Audio download, account login and private playlists are not supported.

## Fetch a playlist

`POST /api/v1/songlists`, `Content-Type: application/json`:

```json
{
  "url": "https://music.163.com/playlist?id=19723756",
  "detailed": false,
  "format": "song-singer",
  "order": "normal"
}
```

The URL can also be the full share text copied from a music app. Supported sources:

The same string `url` field also accepts a numeric Kugou playlist code, for example `{"url":"40725013"}` or `{"url":"酷狗码：40725013"}`. Codes are kept as strings, including leading zeros. A code is resolved through Kugou's official HTTPS command endpoint; a returned public collection or special playlist then uses the existing paginated adapter. Both the JSON and legacy form routes support this input. Invalid/expired codes and codes for non-playlist resources return HTTP 400; an upstream failure remains 502/504. Numeric input is interpreted as a Kugou code, not a raw playlist ID for another platform.

| Source | Links |
| --- | --- |
| `netease` | `music.163.com` playlist/album links, hash routes, `163cn.tv` / `163cn.com` shares |
| `qq` | `y.qq.com/n/ryqq/playlist/ID`, `ryqq_v2`, `i.y.qq.com` / `i2.y.qq.com` detail pages, `c6.y.qq.com` short links |
| `qishui` | `qishui.douyin.com/s/...`, `music.douyin.com/qishui/share/playlist?playlist_id=...` |
| `kugou` | `www.kugou.com/songlist/gcid_...`, official special playlists, Concept Edition `global_specialid=collection_...`, official `t.kugou.com` / `t1.kugou.com` redirects |

For a code response, GoMusic keeps the command metadata's known total even if a later endpoint reports fewer tracks. A complete embedded list can be exported directly; incomplete or unknown-total embedded data remains explicitly partial. No music account credentials are used or accepted.

`detailed=true` preserves original titles; otherwise platform annotations in `【...】` are removed and Chinese parentheses are normalized. `format` supports `song-singer` (default), `singer-song`, and `song`. `order` supports `normal` (default) and `reverse`. Title and artist fields stay separate during formatting, including when they contain ` - `.

```json
{
  "code": 1,
  "msg": "success",
  "data": {
    "name": "My playlist",
    "source": "netease",
    "songs": ["Song - Artist"],
    "tracks": [{"id": "123", "title": "Song", "artists": ["Artist"]}],
    "songs_count": 1,
    "total_count": 1,
    "complete": true,
    "warnings": []
  }
}
```

`tracks` contains original metadata in the requested order. `songs` contains formatted export lines. `songs_count` always equals the number of actually returned tracks; `total_count` preserves the known upstream total. Clients **must check `complete` and display `warnings`** before describing an export as complete. Provider restrictions, missing tracks, capped pages, or the configured song limit can produce a useful partial result with HTTP 200 and `complete=false`.

When a legacy endpoint or share-page fallback reports a smaller playlist total, GoMusic retains the largest total observed across the responses. A 60-track preview cannot turn a known 2,800-track playlist into a complete export. Paginated QQ, Qishui and Kugou results reconcile overlapping pages by stable song ID; different song IDs with the same title remain separate tracks. Pagination stops with a partial-result warning when subsequent pages contain no new valid tracks.

Kugou standard-edition `gcid` share pages sometimes return only a preview. GoMusic reports it as partial when the advertised count is larger. Public Concept Edition collections and official special playlists have separate paginated adapters. Availability remains dependent on the music platform; a successful sample does not guarantee every playlist is publicly accessible.

For an incomplete result taken from the public Kugou share-page preview, `preview=true` distinguishes that limitation from an interrupted paginated request. The adapter tries valid collection IDs (including top-level share metadata) and official special IDs when available, retaining the largest known total. For the observed v3 GCID shape, it can also decode a candidate public collection identity using base-35 fields, require its owner to match the official share metadata, and require every visible preview track ID to match the beginning of the retrieved collection in order. The trailer is not treated as an authenticated identity. Unknown formats, missing/mismatched owners, or mismatched track prefixes keep the preview instead of substituting another playlist.

The original `gcid_3zbivkavz4z072` share is now resolved to `collection_3_606105881_4_0`, retaining its original name and returning 12/12 tracks. When none of the validated public paths succeeds, a page exposing 10 of 12 tracks remains `songs_count=10`, `total_count=12`, `complete=false`; it is never silently described as a full export.

## Legacy API

`POST /songlist?detailed=true&format=singer-song&order=reverse` accepts `application/x-www-form-urlencoded` with a `url` field. The original `{code, msg, data}` envelope remains. Both routes also accept JSON; explicit query options take precedence over JSON options. Music-platform account credentials and cookies are not accepted or forwarded. The optional first-party usage cookie is described below.

## Errors and limits

Errors use `{ "code": -1, "msg": "A readable explanation", "data": null }`.

| HTTP | Meaning |
| --- | --- |
| 400 | Invalid URL, malformed JSON/form, unsupported format/order |
| 403 | A known private NetEase playlist; set it to public first |
| 413 | Request body exceeds 8 KiB |
| 415 | Unsupported content type |
| 502 | Platform unavailable, rejected request or invalid upstream response |
| 504 | Request or upstream timeout |

Share input is limited to 4096 bytes. Only exact official hosts are allowed. Each redirect is revalidated, limited to six hops, and must stay within its music platform. Upstream responses are limited to 16 MiB, each request has a deadline, and NetEase detail batching has bounded concurrency. Playlist export defaults to a 10,000-song cap. Missing data never becomes a falsely complete success.

## Health

- `GET /healthz`: process liveness, HTTP 200.
- `GET /readyz`: configured storage checks, HTTP 200 when healthy or HTTP 503 with `status=degraded`. No connection strings or passwords are returned. Stateless mode has no storage checks.

A storage failure during a playlist request falls back from Redis to MySQL to the upstream service. Readiness still reports the configured dependency failure so operators can repair it.

## Persistent usage statistics

With MySQL configured, both playlist POST routes record usage in `gomusic_usage_daily` and `gomusic_usage_visitors`. These counters have no cache expiry and survive process/container restarts. Existing song-cache tables and Redis keys are retained.

Set `GOMUSIC_STATS_TOKEN` to enable the private `GET /api/v1/stats` endpoint and send `Authorization: Bearer <token>`. Without a token the endpoint is disabled; it never exposes visitor identifiers. For local review:

```sh
curl -H "Authorization: Bearer $GOMUSIC_STATS_TOKEN" http://localhost:8081/api/v1/stats
curl -H "Authorization: Bearer $GOMUSIC_STATS_TOKEN" 'http://localhost:8081/api/v1/stats?period=week&date=2026-11-15'
```

Without query parameters, `data.periods` contains `day`, `week`, `month`, and `all`. Each entry contains `start_date`, `end_date` (inclusive), `requests`, `successes`, `failures`, `partial_results`, `songs`, and `unique_visitors`. Optional `period=day|week|month|all` selects one period; optional `date=YYYY-MM-DD` selects a historical calendar date. All periods use Asia/Shanghai; weeks begin on Monday. `all` means all persisted usage through the selected date, starting from collection activation rather than the latest process restart.

`songs` sums tracks actually returned, including useful partial results. Repeated retrievals add songs again; format changes, copying, and TXT downloads do not count as new retrievals. `unique_visitors` counts distinct anonymous browser/client identifiers across the whole requested period, not a sum of daily visitor counts. A first-party HttpOnly, SameSite=Lax cookie provides the identifier; only its SHA-256 hash and calendar date are stored. Clearing cookies, changing devices, or an API client not retaining cookies can count as another visitor. No IP addresses or playlist links are stored in these statistics. A failed metrics write is logged and does not turn a successful playlist response into an error.

For HTTPS deployments, set `GOMUSIC_STATS_COOKIE_SECURE=true`. Same-origin frontend/API deployment is recommended. For a separate API origin, configure the exact frontend origin in `GOMUSIC_ALLOWED_ORIGINS`; wildcards, paths, and embedded credentials are rejected. The frontend includes cookies and the server permits credentials only for configured origins. Use same-site frontend/API hosts (for example `music.example.com` and `api.example.com`) with HTTPS; unrelated sites remain subject to SameSite and third-party-cookie restrictions.
