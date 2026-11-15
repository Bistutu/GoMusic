# Third-party references

GoMusic's original LICENSE remains in force. Provider adapters were independently rewritten in Go after protocol research. The following MIT sources informed the protocol/signature and parsing work; no cross-repository runtime dependencies were added.

- [PlaylistOut](https://github.com/LengxiQwQ/playlistout), commit `61d148f839cc7e7829c15df6a1c36dfa42fa9690` — Copyright (c) 2026 LengxiQwQ.
- [Meting](https://github.com/metowolf/Meting) — Copyright (c) 2018 METO <i@i-meto.com>.
- [KuGouMusicApi](https://github.com/MakcRe/KuGouMusicApi) — Copyright (c) 2023 MakcRe.

MIT License

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

Installed dependencies retain their own licenses in Go modules and npm packages. See `go.mod`, `go.sum` and `static/package-lock.json` for exact dependency versions.

Additional protocol-only research for numeric Kugou codes used LX Music (`lyswhut/lx-music-desktop`, commit `323e4bac505da428adae8db6b58039fb2b36b86d`) and wp_MusicApi (`GitHub-ZC/wp_MusicApi`, commit `bf9307dd138dc8ac6c4f7de29361209d4f5b665f`), both Apache-2.0 licensed. The Go implementation is independent; no source code from these projects is incorporated. Links and the verified official protocol are recorded in `docs/REFERENCES.md`.
