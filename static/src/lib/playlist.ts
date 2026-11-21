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

export const playlistSources = [
  "netease",
  "qq",
  "qishui",
  "kugou",
  "kuwo",
  "bodian",
  "migu",
  "qianqian",
] as const;
type PlaylistSource = (typeof playlistSources)[number];

const hosts: Partial<Record<string, PlaylistSource>> = {
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
  "beta-luna.douyin.com": "qishui",
  "kuwo.cn": "kuwo",
  "www.kuwo.cn": "kuwo",
  "m.kuwo.cn": "kuwo",
  "t.kuwo.cn": "kuwo",
  "h5app.kuwo.cn": "bodian",
  "bodian.kuwo.cn": "bodian",
  "music.migu.cn": "migu",
  "music.91q.com": "qianqian",
};
export function platformOf(input: string): PlaylistSource | undefined {
  const escapedQuerySeparators = /\\u0026/gi;
  const inputs = [input];
  for (let depth = 0; depth < 2; depth++) {
    try {
      const decoded = decodeURIComponent(
        inputs[depth].replace(/%(?![0-9a-f]{2})/gi, "%25"),
      );
      if (decoded === inputs[depth]) break;
      inputs.push(decoded);
    } catch {
      break;
    }
  }
  for (const rawInput of inputs) {
    const candidateInput = rawInput
      .replace(/\\\//g, "/")
      .replace(escapedQuerySeparators, "&");
    const candidates: Array<{ start: number; value: string }> = [
      ...candidateInput.matchAll(
        /https?:\/\/[^\s<>"'\u3000\u200b，。；！、（）【】《》〉,;!()\]]+/gi,
      ),
    ].map((match) => ({ start: match.index ?? 0, value: match[0] }));
    for (const match of candidateInput.matchAll(
      /(?:^|[^a-z0-9@./:-])(\/\/[a-z0-9-]+(?:\.[a-z0-9-]+)+\/[^\s<>"'，。；！、）】》〉,;!)]+)/gi,
    )) {
      candidates.push({
        start: (match.index ?? 0) + match[0].indexOf(match[1]),
        value: `https:${match[1]}`,
      });
    }
    for (const match of candidateInput.matchAll(
      /(?:^|[^a-z0-9@./-])([a-z0-9-]+(?:\.[a-z0-9-]+)+\/[^\s<>"'，。；！、）】》〉,;!)]+)/gi,
    )) {
      candidates.push({
        start: (match.index ?? 0) + match[0].indexOf(match[1]),
        value: `https://${match[1]}`,
      });
    }
    candidates.sort((a, b) => a.start - b.start);
    for (const { value: candidate } of candidates) {
      try {
        const url = new URL(
          candidate.replace(/[，。；！、）】》〉,;!)\u200b]+$/, ""),
        );
        if (!url.username && !url.password && !url.port && hosts[url.hostname])
          return hosts[url.hostname];
      } catch {
        /* Continue to the next link in share text. */
      }
    }
    if (kugouCodeOf(candidateInput)) return "kugou";
  }
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
