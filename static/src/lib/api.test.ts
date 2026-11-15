import { afterEach, expect, it, vi } from "vitest";
import { fetchPlaylist, PlaylistError } from "./api";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
it("handles a failed connection without dereferencing a nonexistent response", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockRejectedValue(new TypeError("Failed to fetch")),
  );
  await expect(fetchPlaylist("url")).rejects.toMatchObject({ kind: "network" });
});
it("rejects malformed successful responses", async () => {
  vi.stubGlobal(
    "fetch",
    vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({ code: 1, data: {} }))),
  );
  await expect(fetchPlaylist("url")).rejects.toBeInstanceOf(PlaylistError);
});
it("preserves a structured server error", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ code: -1, msg: "Unavailable" }), {
        status: 502,
      }),
    ),
  );
  await expect(fetchPlaylist("url")).rejects.toMatchObject({
    kind: "server",
    message: "Unavailable",
  });
});
it.each([
  [400, "invalid"],
  [504, "timeout"],
])("classifies HTTP %i as %s", async (status, kind) => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ code: -1, msg: "Request failed" }), {
        status: Number(status),
      }),
    ),
  );
  await expect(fetchPlaylist("url")).rejects.toMatchObject({ kind });
});
it("aborts an in-flight request when cancelled", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(
      (_url, options) =>
        new Promise((_resolve, reject) =>
          options.signal.addEventListener("abort", () =>
            reject(new DOMException("Aborted", "AbortError")),
          ),
        ),
    ),
  );
  const controller = new AbortController();
  const request = fetchPlaylist("url", controller.signal);
  controller.abort();
  await expect(request).rejects.toMatchObject({ name: "AbortError" });
});

it("retains the anonymous visitor cookie when using a separate API origin", async () => {
  const fetch = vi.fn().mockResolvedValue(
    new Response(
      JSON.stringify({
        code: 1,
        data: { tracks: [], total_count: 0, complete: true, warnings: [] },
      }),
    ),
  );
  vi.stubGlobal("fetch", fetch);
  await fetchPlaylist("https://music.163.com/playlist?id=1");
  expect(fetch.mock.calls[0]?.[1]).toMatchObject({ credentials: "include" });
});
