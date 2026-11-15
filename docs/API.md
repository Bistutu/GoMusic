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

| Source | Links |
| --- | --- |
| `netease` | `music.163.com` playlist/album links, hash routes, `163cn.tv` / `163cn.com` shares |
| `qq` | `y.qq.com/n/ryqq/playlist/ID`, `ryqq_v2`, `i.y.qq.com` / `i2.y.qq.com` detail pages, `c6.y.qq.com` short links |
| `qishui` | `qishui.douyin.com/s/...`, `music.douyin.com/qishui/share/playlist?playlist_id=...` |
| `kugou` | `www.kugou.com/songlist/gcid_...`, official special playlists, Concept Edition `global_specialid=collection_...`, official `t.kugou.com` / `t1.kugou.com` redirects |

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

Kugou standard-edition `gcid` share pages sometimes return only a preview. GoMusic reports it as partial when the advertised count is larger. Public Concept Edition collections and official special playlists have separate paginated adapters. Availability remains dependent on the music platform; a successful sample does not guarantee every playlist is publicly accessible.

## Legacy API

`POST /songlist?detailed=true&format=singer-song&order=reverse` accepts `application/x-www-form-urlencoded` with a `url` field. The original `{code, msg, data}` envelope remains. Both routes also accept JSON; explicit query options take precedence over JSON options. No credentials or cookies are accepted or forwarded.

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
