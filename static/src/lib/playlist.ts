export interface Track {
  id: string;
  title: string;
  artists: string[];
}
export interface Playlist {
  name: string;
  source: string;
  songs: string[];
  tracks: Track[];
  songs_count: number;
  total_count: number;
  complete: boolean;
  warnings: string[];
  preview?: boolean;
}
export interface ExportOptions {
  detailed: boolean;
  format: "song-singer" | "singer-song" | "song";
  order: "normal" | "reverse";
}

const hosts: Record<string, string> = {
  "music.163.com": "netease",
  "y.music.163.com": "netease",
  "163cn.tv": "netease",
  "163cn.com": "netease",
  "y.qq.com": "qq",
  "i.y.qq.com": "qq",
  "i2.y.qq.com": "qq",
  "c6.y.qq.com": "qq",
  "c.y.qq.com": "qq",
  "u.y.qq.com": "qq",
  "u6.y.qq.com": "qq",
  "kugou.com": "kugou",
  "www.kugou.com": "kugou",
  "m.kugou.com": "kugou",
  "t.kugou.com": "kugou",
  "t1.kugou.com": "kugou",
  "activity.kugou.com": "kugou",
  "qishui.douyin.com": "qishui",
  "music.douyin.com": "qishui",
};
export function platformOf(input: string): string | undefined {
  for (const match of input.matchAll(/https?:\/\/[^\s<>"\u3000]+/g)) {
    try {
      const url = new URL(
        match[0].replace(/[，。；！、）】》〉,;!)\u200b]+$/, ""),
      );
      if (!url.username && !url.password && !url.port && hosts[url.hostname])
        return hosts[url.hostname];
    } catch {
      /* Continue to the next link in share text. */
    }
  }
  if (kugouCodeOf(input)) return "kugou";
}
export function kugouCodeOf(input: string): string | undefined {
  return /^(?:(?:酷狗码|kugou\s+code)\s*[:：]?\s*)?([0-9]{1,20})$/i.exec(
    input.trim(),
  )?.[1];
}
export function standardTitle(title: string): string {
  return title
    .replace(/（/g, " (")
    .replace(/）/g, ")")
    .replace(/\s?【[^】]*】/g, "")
    .trim();
}
export function formatTracks(
  tracks: Track[],
  options: ExportOptions,
): string[] {
  const songs = tracks.map((track) => {
    const title = options.detailed ? track.title : standardTitle(track.title);
    const artist = track.artists.join(" / ");
    if (options.format === "song" || !artist) return title;
    return options.format === "singer-song"
      ? `${artist} - ${title}`
      : `${title} - ${artist}`;
  });
  return options.order === "reverse" ? songs.reverse() : songs;
}
export function safeFilename(name: string): string {
  return (
    (name
      .replace(/[<>:"/\\|?*\u0000-\u001f]/g, "_")
      .trim()
      .slice(0, 100) || "GoMusic") + ".txt"
  );
}
export async function copyText(text: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      /* Use selection fallback. */
    }
  }
  const previous = document.activeElement;
  const area = document.createElement("textarea");
  area.value = text;
  area.readOnly = true;
  area.style.cssText = "position:fixed;left:-9999px;top:0;";
  document.body.append(area);
  area.select();
  const copied = document.execCommand("copy");
  area.remove();
  if (previous instanceof HTMLElement) previous.focus({ preventScroll: true });
  if (!copied) throw new Error("Clipboard unavailable");
}
export function downloadText(name: string, text: string): void {
  const url = URL.createObjectURL(
    new Blob(["\ufeff", text], { type: "text/plain;charset=utf-8" }),
  );
  const a = document.createElement("a");
  a.href = url;
  a.download = safeFilename(name);
  document.body.append(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
