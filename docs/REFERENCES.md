# Implementation and open-source references

The refactor preserves GoMusic's small public-playlist-to-text workflow. Provider code is independently implemented in Go against the observed public protocols; the project does not run or depend on any reference repository or third-party hosted music API.

## Provider research (2026-11-15)

| Reference | Evidence read | What informed this implementation |
| --- | --- | --- |
| [NeteaseCloudMusicApiEnhanced](https://github.com/NeteaseCloudMusicApiEnhanced/api-enhanced/blob/main/module/playlist_track_all.js) | File blob `06758f512e579936a550d2fdfd02385af4d882df` | Use playlist `trackIds`, not only the first embedded tracks; fetch missing song details in batches. GoMusic also reuses the fresh metadata already returned, bounds concurrency, preserves order and caches raw fields. |
| [QQMusicApi documentation](https://github.com/L-1124/QQMusicApi/blob/main/docs/coding.md), [qqmusic-api playlist protocol](https://github.com/ylw1997/qqmusic-api/blob/master/docs/apis/get-playlist-detail.md) | Public request/pagination documentation | Validate business status and advance by actual returned records. Preserve expected totals; detect stalled pages and use the repository's existing legacy fallback. |
| [Meting Kugou provider](https://github.com/metowolf/Meting/blob/master/src/providers/kugou.js) | File blob `b815391ce542c7b7e76116256cc63e9d84fd09f4` | Official special-playlist song endpoint and mobile request conventions. GoMusic paginates and checks counts instead of assuming `pagesize=-1` returns everything. |
| [KuGouMusicApi](https://github.com/MakcRe/KuGouMusicApi/blob/main/module/playlist_track_all.js) | File blob `f8f6fb98ff3eef42affeb773dc471021cccd3624` | Collection IDs and separate playlist detail/song routes. No login or encrypted account operations were imported. |
| [PlaylistOut](https://github.com/LengxiQwQ/playlistout/tree/61d148f839cc7e7829c15df6a1c36dfa42fa9690/worker/src/providers) | Commit `61d148f839cc7e7829c15df6a1c36dfa42fa9690`; Qishui/Kugou/QQ clients and Kugou input/normalization | Qishui's public cursor API; Kugou standard-edition preview versus public collection pagination; public H5 signature parameters. Implemented with Go standard JSON/HTTP/MD5, no provider JavaScript execution. |

The Qishui share-page shape was also verified directly against the issue #20 public example: `_ROUTER_DATA.loaderData.playlist_page`. The current primary path uses `/luna/playlist/detail`; the share page is a fallback. Cursor, response and song limits are enforced by GoMusic.

### Observed Kugou GCID mapping

GoMusic independently checked the public pairs `gcid_3z17seve8z2z051` → `collection_3_2248574973_2_0` and `gcid_3zbivkavz4z072` → `collection_3_606105881_4_0`. The known v3 zero-tag shape separates base-35 fields with `z`; its owner and list fields reproduce those IDs. This is an inference verified against these public samples, not a claim that every GCID format is reversible or that the trailer has been authenticated.

The adapter requires the decoded owner to equal the official share-page `list_create_userid` and requires the retrieved collection's entire visible song prefix to match by stable hash before accepting it. The user's original [public share](https://m.kugou.com/songlist/gcid_3zbivkavz4z072/) exposes 10 previews but its validated [public collection](https://activity.kugou.com/share/index.html?global_specialid=collection_3_606105881_4_0) returns 12. This additional path does not require a login or use a third-party hosted service. Failed validation retains the partial-result warning.

Sources containing referenced nontrivial protocol/signature logic are MIT licensed; relevant notices are retained in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md). No changes were made to FluentRead or its reference repositories.

## Frontend choice

[Element Plus](https://element-plus.org/en-US/guide/design.html), [Reka UI](https://reka-ui.com/) and retaining Vue with native controls were compared for this form/preview page. Element Plus was globally loaded in the old application. Reka offers unstyled accessible components but this page needs only ordinary forms, selects, checkboxes and disclosure controls.

The chosen implementation keeps Vue, replaces Vue CLI with [Vite](https://vite.dev/guide/), uses TypeScript and native controls, and removes the global UI library, Axios and the global `ResizeObserver` override. It uses system fonts, same-origin API calls, and no external advertising script. The original GitHub Star badge, migration instructions, support QR and supporter list remain visible.

Actual browser checks cover initial/loading/result/error states, download contents, cancellation/retry, 500-track batches, preferences, and desktop/mobile overflow. Live screenshot inspection additionally reads a real public playlist and verifies that clipboard contents match the export. The final design follows the owner’s review: a white single-column page, black/gray text and a pale-red accent on primary buttons. The original migration heading is retained with Kugou added. Original-name, format and order controls are visible before fetching; the result area is always present. Help, the original support QR and supporter list are shown without collapsed panels. A retrieved playlist title is labeled “Playlist name” so it cannot be mistaken for an application mode. Initial input remains empty; example playlist names are never hardcoded into the product UI. The GitHub Star badge and Chinese/English switching are preserved.

## Structure

- `internal/config`: explicit environment configuration.
- `internal/musicurl`: official-host validation, share extraction and provider IDs.
- `logic.Service`: provider adapters and common raw metadata/export contract.
- `repo`: optional raw metadata repositories, with request-scoped deadlines and expiration.
- `handler`: legacy/versioned contracts, input limits, health and static frontend.
- `static/src`: Vue components, fetch/error boundary, export utilities and local formatting.

Legacy cache tables/keys are not rewritten. New raw metadata uses `song_metadata` and `gomusic:v2:` Redis keys. Titles are not unique, and presentation choices never contaminate stored metadata.
