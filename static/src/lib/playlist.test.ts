import { describe, expect, it } from "vitest";
import { formatTracks, platformOf, safeFilename } from "./playlist";

it("recognizes share text and rejects lookalike hosts", () => {
  expect(
    platformOf("歌单 https://qishui.douyin.com/s/iAA9MFsL/ @汽水音乐"),
  ).toBe("qishui");
  expect(
    platformOf(
      "https://music.douyin.com/qishui/share/playlist?auto_play_bgm=1&playlist_id=7203983736064622592",
    ),
  ).toBe("qishui");
  expect(
    platformOf(
      '{"url":"https:\\/\\/music.douyin.com\\/qishui\\/share\\/playlist?auto_play_bgm=1\\u0026playlist_id=7203983736064622592"}',
    ),
  ).toBe("qishui");
  expect(platformOf("https://www.kugou.com/songlist/gcid_example/")).toBe(
    "kugou",
  );
  for (const url of [
    "https://y.qq.com.evil.test",
    "https://y.qq.com@127.0.0.1",
    "https://evil.test/qq/163",
    "https://music.163.com:8080/",
    "https://t.kuwo.cn.evil.test/AbC123",
    "https%3A%2F%2Fevil.test%2Fplaylist%2F1",
  ])
    expect(platformOf(url)).toBeUndefined();
});
it("detects the expanded Chinese providers and link formats", () => {
  expect(platformOf("www.kuwo.cn/playlist_detail/236289985")).toBe("kuwo");
  expect(platformOf("酷我歌单 https://t.kuwo.cn/AbC123")).toBe("kuwo");
  expect(
    platformOf(
      "歌单有效率100% https%3A%2F%2Fwww.kuwo.cn%2Fplaylist_detail%2F236289985",
    ),
  ).toBe("kuwo");
  expect(
    platformOf(
      "https%253A%252F%252Fmusic.163.com%252Fplaylist%253Fid%253D8725919816",
    ),
  ).toBe("netease");
  expect(
    platformOf("HTTP://music.migu.cn/v5/#/playlist?playlistId=231760782"),
  ).toBe("migu");
  expect(platformOf("//music.91q.com/songlist/295893")).toBe("qianqian");
  expect(
    platformOf(
      "https://h5app.kuwo.cn/m/bodian/collection.html?playlistId=86749638&source=5",
    ),
  ).toBe("bodian");
  expect(
    platformOf(
      "https://h5app.kuwo.cn/m/bodian/collection.html?playlistId=86749638&source=5",
    ),
  ).toBe("bodian");
  expect(
    platformOf(
      "https://beta-luna.douyin.com/luna/playlist/detail?playlist_id=1",
    ),
  ).toBe("qishui");
  expect(platformOf("https://www.deezer.com/en/playlist/5339620562")).toBe(
    "deezer",
  );
  expect(platformOf("https://deezer.page.link/abc123")).toBe("deezer");
  expect(platformOf("https://dzr.page.link/abc123")).toBe("deezer");
  expect(
    platformOf("https://deezer.page.link.evil.test/abc123"),
  ).toBeUndefined();
  expect(platformOf("https://link.deezer.com/s/32zD0iv1H70Dypq0V2AHe")).toBe(
    "deezer",
  );
  expect(platformOf("https://api.deezer.com/playlist/5339620562")).toBe(
    undefined,
  );
});
describe("export formatting", () => {
  const tracks = [
    {
      id: "1",
      title: "Song - Version（Live）【tag】",
      artists: ["Artist - Name", "Other"],
    },
    { id: "2", title: "Another", artists: [] },
  ];
  it("preserves title and artist hyphens without mutating source order", () => {
    expect(
      formatTracks(tracks, {
        detailed: false,
        format: "singer-song",
        order: "reverse",
      }),
    ).toEqual(["Another", "Artist - Name / Other - Song - Version (Live)"]);
    expect(tracks[0].id).toBe("1");
  });
  it("exports original names and song-only format", () => {
    expect(
      formatTracks(tracks, {
        detailed: true,
        format: "song",
        order: "normal",
      })[0],
    ).toBe(tracks[0].title);
  });
  it("creates usable filenames", () => {
    expect(safeFilename("../A/B:List")).toBe(".._A_B_List.txt");
    expect(safeFilename("")).toBe("GoMusic.txt");
  });
});
