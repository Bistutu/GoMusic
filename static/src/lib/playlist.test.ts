import { describe, expect, it } from "vitest";
import { formatTracks, platformOf, safeFilename } from "./playlist";

it("recognizes share text and rejects lookalike hosts", () => {
  expect(
    platformOf("歌单 https://qishui.douyin.com/s/iAA9MFsL/ @汽水音乐"),
  ).toBe("qishui");
  expect(platformOf("https://www.kugou.com/songlist/gcid_example/")).toBe(
    "kugou",
  );
  for (const url of [
    "https://y.qq.com.evil.test",
    "https://y.qq.com@127.0.0.1",
    "https://evil.test/qq/163",
    "https://music.163.com:8080/",
  ])
    expect(platformOf(url)).toBeUndefined();
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
