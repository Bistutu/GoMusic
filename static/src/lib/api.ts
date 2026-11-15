import type { Playlist } from "./playlist";

export class PlaylistError extends Error {
  constructor(
    public kind: "invalid" | "network" | "timeout" | "server" | "response",
    message = "",
  ) {
    super(message);
  }
}

export async function fetchPlaylist(
  url: string,
  signal?: AbortSignal,
): Promise<Playlist> {
  const controller = new AbortController();
  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, 100_000);
  const abort = () => controller.abort();
  signal?.addEventListener("abort", abort, { once: true });
  if (signal?.aborted) controller.abort();
  try {
    const response = await fetch(
      `${import.meta.env.VITE_API_BASE_URL || ""}/api/v1/songlists`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url, detailed: true }),
        signal: controller.signal,
      },
    );
    let body;
    try {
      body = await response.json();
    } catch {
      throw new PlaylistError("response");
    }
    if (!response.ok || body?.code !== 1)
      throw new PlaylistError(
        response.status === 504
          ? "timeout"
          : response.status === 400
            ? "invalid"
            : "server",
        typeof body?.msg === "string" ? body.msg : "",
      );
    const data = body.data;
    if (
      !data ||
      !Array.isArray(data.tracks) ||
      !data.tracks.every((t: unknown) => {
        if (typeof t !== "object" || t === null) return false;
        const track = t as Record<string, unknown>;
        return (
          typeof track.title === "string" &&
          Array.isArray(track.artists) &&
          track.artists.every((a) => typeof a === "string")
        );
      }) ||
      !Number.isInteger(data.total_count) ||
      typeof data.complete !== "boolean" ||
      !Array.isArray(data.warnings)
    ) {
      throw new PlaylistError("response");
    }
    return data as Playlist;
  } catch (error) {
    if (timedOut) throw new PlaylistError("timeout");
    if (signal?.aborted) throw new DOMException("Cancelled", "AbortError");
    if (error instanceof PlaylistError) throw error;
    throw new PlaylistError("network");
  } finally {
    clearTimeout(timer);
    signal?.removeEventListener("abort", abort);
  }
}
