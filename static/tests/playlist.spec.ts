import { expect, test } from "@playwright/test";

const playlist = (n = 2, complete = true) => ({
  code: 1,
  msg: "success",
  data: {
    name: "我的好音乐",
    source: "qq",
    tracks: Array.from({ length: n }, (_, i) => ({
      id: `${i + 1}`,
      title: `歌曲 ${i + 1} - Live【平台注释】`,
      artists: ["歌手甲", "歌手乙"],
    })),
    songs: [],
    songs_count: n,
    total_count: complete ? n : n + 30,
    complete,
    warnings: complete ? [] : ["partial"],
  },
});

test("keeps the page usable on desktop and mobile", async ({
  page,
}, testInfo) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "歌单导出与迁移",
  );
  const sources = page.getByRole("list", { name: "支持的公开歌单来源" });
  await expect(sources).toBeInViewport({ ratio: 1 });
  await expect(sources.getByRole("listitem")).toHaveText([
    "网易云音乐",
    "QQ 音乐",
    "汽水音乐",
    "酷狗音乐",
    "酷我音乐",
    "波点音乐",
    "咪咕音乐",
    "千千音乐",
    "Deezer",
  ]);
  const sourceBounds = (await sources.boundingBox())!;
  const description = page.getByText("复制歌曲列表或下载 TXT", {
    exact: false,
  });
  const input = page.getByLabel("公开歌单链接或酷狗码");
  expect(sourceBounds.y + sourceBounds.height).toBeLessThan(
    (await description.boundingBox())!.y,
  );
  expect(sourceBounds.y + sourceBounds.height).toBeLessThan(
    (await input.boundingBox())!.y,
  );
  await expect(
    page.getByRole("button", { name: "获取歌单", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "复制结果", exact: true }),
  ).toBeDisabled();
  await expect(
    page.getByRole("checkbox", { name: "保留原始歌名" }),
  ).toBeVisible();
  await expect(
    page.getByRole("radio", { name: "歌手 - 歌名", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("radio", { name: "倒序", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "使用指南", exact: true }),
  ).toBeVisible();
  await expect(page.getByAltText("WeChat")).toBeVisible();
  await expect(page.getByText("冰楠 ₿.Ξ 🧘", { exact: true })).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: `test-results/${testInfo.project.name}-initial.png`,
    fullPage: true,
  });
  expect(errors).toEqual([]);
});
test("validates links and shows useful network errors", async ({ page }) => {
  await page.goto("/");
  await page
    .getByLabel("公开歌单链接或酷狗码")
    .fill("https://y.qq.com.evil.test/playlist/1");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("支持平台");
  await page.route("**/api/v1/songlists", (route) => route.abort("failed"));
  await page
    .getByLabel("公开歌单链接或酷狗码")
    .fill("https://y.qq.com/n/ryqq/playlist/1");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("连接失败");
});
test("accepts a Kuwo short share link in the shared input flow", async ({
  page,
}, testInfo) => {
  let submittedUrl = "";
  await page.route("**/api/v1/songlists", async (route) => {
    submittedUrl = route.request().postDataJSON().url;
    await route.fulfill({
      json: {
        ...playlist(2),
        data: {
          ...playlist(2).data,
          name: "Kuwo short-link playlist",
          source: "kuwo",
        },
      },
    });
  });
  await page.goto("/");
  const shareText = "酷我歌单 https://t.kuwo.cn/AbC123";
  await page.getByLabel("公开歌单链接或酷狗码").fill(shareText);
  await expect(page.getByText("已识别平台：酷我音乐")).toBeVisible();
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.locator(".playlist-name")).toContainText(
    "Kuwo short-link playlist",
  );
  await expect(page.getByLabel("可编辑的歌曲文本")).toHaveValue(/歌曲 1/);
  expect(submittedUrl).toBe(shareText);
  await page.screenshot({
    path: `test-results/${testInfo.project.name}-kuwo-shortlink.png`,
    fullPage: true,
  });
});
test("accepts a percent-encoded Kuwo link in the shared input flow", async ({
  page,
}, testInfo) => {
  let submittedUrl = "";
  await page.route("**/api/v1/songlists", async (route) => {
    submittedUrl = route.request().postDataJSON().url;
    await route.fulfill({
      json: {
        ...playlist(2),
        data: {
          ...playlist(2).data,
          name: "Kuwo encoded-link playlist",
          source: "kuwo",
        },
      },
    });
  });
  await page.goto("/");
  const encodedUrl = "https%3A%2F%2Fwww.kuwo.cn%2Fplaylist_detail%2F236289985";
  await page.getByLabel("公开歌单链接或酷狗码").fill(encodedUrl);
  await expect(page.getByText("已识别平台：酷我音乐")).toBeVisible();
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.locator(".playlist-name")).toContainText(
    "Kuwo encoded-link playlist",
  );
  expect(submittedUrl).toBe(encodedUrl);
  await page.screenshot({
    path: `test-results/${testInfo.project.name}-kuwo-encoded-link.png`,
    fullPage: true,
  });
});
test("recognizes JSON-escaped share links without rewriting the input", async ({
  page,
}) => {
  let submittedUrl = "";
  await page.route("**/api/v1/songlists", async (route) => {
    submittedUrl = route.request().postDataJSON().url;
    await route.fulfill({
      json: {
        ...playlist(2),
        data: {
          ...playlist(2).data,
          name: "Qishui JSON share",
          source: "qishui",
        },
      },
    });
  });
  await page.goto("/");
  const shareText =
    '{"url":"https:\\/\\/music.douyin.com\\/qishui\\/share\\/playlist?auto_play_bgm=1\\u0026playlist_id=7203983736064622592"}';
  await page.getByLabel("公开歌单链接或酷狗码").fill(shareText);
  await expect(page.getByText("已识别平台：汽水音乐")).toBeVisible();
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.locator(".playlist-name")).toContainText(
    "Qishui JSON share",
  );
  expect(submittedUrl).toBe(shareText);
});
test("accepts a Bodian public playlist share link", async ({
  page,
}, testInfo) => {
  await page.route("**/api/v1/songlists", (route) =>
    route.fulfill({
      json: {
        ...playlist(2),
        data: {
          ...playlist(2).data,
          name: "Bodian playlist",
          source: "bodian",
        },
      },
    }),
  );
  await page.goto("/");
  await page
    .getByLabel("公开歌单链接或酷狗码")
    .fill(
      "https://h5app.kuwo.cn/m/bodian/collection.html?uid=45780003&playlistId=86749638&source=5&owerId=",
    );
  await expect(page.getByText("已识别平台：波点音乐")).toBeVisible();
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.locator(".playlist-name")).toContainText("Bodian playlist");
  await expect(page.getByLabel("可编辑的歌曲文本")).toHaveValue(/歌曲 1/);
  await page.screenshot({
    path: `test-results/${testInfo.project.name}-bodian.png`,
    fullPage: true,
  });
});
test("recognizes a Deezer short share link in the shared input", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByLabel("公开歌单链接或酷狗码")
    .fill("https://link.deezer.com/s/32zD0iv1H70Dypq0V2AHe");
  await expect(page.getByText("已识别平台：Deezer")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "获取歌单", exact: true }),
  ).toBeEnabled();
});

test("explains retired Deezer links in both languages", async ({
  page,
}, testInfo) => {
  const shareURL = "https://deezer.page.link/kHcRMSzZEddVF4dd9";
  let receivedURL = "";
  await page.route("**/api/v1/songlists", async (route) => {
    receivedURL = route.request().postDataJSON().url;
    await route.fulfill({
      status: 400,
      json: {
        code: -1,
        msg: "Deezer 旧版 page.link 短链已停止跳转，请重新复制当前分享链接",
        data: null,
      },
    });
  });
  await page.goto("/");
  await page.getByLabel("公开歌单链接或酷狗码").fill(shareURL);
  await expect(page.getByText("已识别平台：Deezer")).toBeVisible();
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("旧版分享链接已失效");
  await expect(page.getByRole("alert")).toContainText(
    "重新复制当前公开歌单分享链接",
  );
  expect(receivedURL).toBe(shareURL);
  await page.screenshot({
    path: `test-results/${testInfo.project.name}-deezer-retired-link.png`,
    fullPage: true,
  });
  await page.getByRole("button", { name: "Switch to English" }).click();
  await expect(page.getByRole("alert")).toContainText(
    "This old Deezer share link no longer works",
  );
});

test("formats locally, reverses, edits and downloads the selected text", async ({
  page,
}, testInfo) => {
  let requests = 0;
  await page.route("**/api/v1/songlists", (route) => {
    requests++;
    return route.fulfill({ json: playlist() });
  });
  await page.goto("/");
  await page
    .getByLabel("公开歌单链接或酷狗码")
    .fill("分享歌单 https://y.qq.com/n/ryqq/playlist/1");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  const output = page.getByLabel("可编辑的歌曲文本");
  await expect(output).toHaveValue(
    "歌曲 1 - Live - 歌手甲 / 歌手乙\n歌曲 2 - Live - 歌手甲 / 歌手乙",
  );
  await page.getByRole("radio", { name: "歌手 - 歌名", exact: true }).check();
  await page.getByRole("radio", { name: "倒序", exact: true }).check();
  await expect(output).toHaveValue(
    "歌手甲 / 歌手乙 - 歌曲 2 - Live\n歌手甲 / 歌手乙 - 歌曲 1 - Live",
  );
  await page.getByLabel("保留原始歌名").check();
  expect(await output.inputValue()).toContain("【平台注释】");
  expect(requests).toBe(1);
  await output.fill("手动编辑的歌曲");
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "下载 TXT" }).click();
  const file = await download;
  expect(file.suggestedFilename()).toBe("我的好音乐.txt");
  const stream = await file.createReadStream();
  const chunks: Buffer[] = [];
  for await (const chunk of stream!) chunks.push(Buffer.from(chunk));
  expect(Buffer.concat(chunks).toString("utf8")).toBe("\ufeff手动编辑的歌曲");
  await page.screenshot({
    path: `test-results/${testInfo.project.name}-result.png`,
    fullPage: true,
  });
});
test("shows incomplete counts and supports a final 1-track batch", async ({
  page,
}) => {
  await page.route("**/api/v1/songlists", (route) =>
    route.fulfill({ json: playlist(501, false) }),
  );
  await page.goto("/");
  await page
    .getByLabel("公开歌单链接或酷狗码")
    .fill("https://y.qq.com/n/ryqq/playlist/1");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByText("完整性未确认", { exact: true })).toBeVisible();
  await expect(page.getByText("平台显示 531", { exact: false })).toBeVisible();
  const batches = page.getByRole("radiogroup", { name: "分批导出" });
  const all = batches.getByRole("radio", { name: "全部歌曲", exact: true });
  const split = batches.getByRole("radio", {
    name: "每批 500 首",
    exact: true,
  });
  await expect(all).toBeVisible();
  await expect(split).toBeVisible();
  await expect(all).toBeChecked();
  await split.check();
  expect(
    (await page.getByLabel("可编辑的歌曲文本").inputValue()).split("\n"),
  ).toHaveLength(500);
  await page.getByRole("button", { name: "下一批" }).click();
  expect(await page.getByLabel("可编辑的歌曲文本").inputValue()).toBe(
    "歌曲 501 - Live - 歌手甲 / 歌手乙",
  );
  await all.check();
  expect(
    (await page.getByLabel("可编辑的歌曲文本").inputValue()).split("\n"),
  ).toHaveLength(501);
  await expect(page.getByRole("button", { name: "下一批" })).not.toBeVisible();
  await all.focus();
  await page.keyboard.press("ArrowRight");
  await expect(split).toBeChecked();
  expect(
    (await page.getByLabel("可编辑的歌曲文本").inputValue()).split("\n"),
  ).toHaveLength(500);
  await expect(page.getByRole("button", { name: "上一批" })).toBeDisabled();
});
test("discloses a short response from a large playlist", async ({ page }) => {
  const body = playlist(30, false);
  body.data.total_count = 2800;
  await page.route("**/api/v1/songlists", (route) =>
    route.fulfill({ json: body }),
  );
  await page.goto("/");
  await page.getByRole("button", { name: "填入范例链接", exact: true }).click();
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByText("完整性未确认", { exact: true })).toBeVisible();
  await expect(page.locator(".result-heading")).toContainText(
    "已获取 30 首歌曲",
  );
  await expect(page.getByText("平台显示 2800", { exact: false })).toBeVisible();
  const output = page.getByLabel("可编辑的歌曲文本");
  expect((await output.inputValue()).split("\n")).toHaveLength(30);
  await page.getByRole("radio", { name: "倒序", exact: true }).check();
  expect((await output.inputValue()).split("\n")[0]).toContain("歌曲 30");
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "下载 TXT", exact: true }).click();
  const stream = await (await download).createReadStream();
  const chunks: Buffer[] = [];
  for await (const chunk of stream!) chunks.push(Buffer.from(chunk));
  const text = Buffer.concat(chunks)
    .toString("utf8")
    .replace(/^\ufeff/, "");
  expect(text).toBe(await output.inputValue());
  expect(text.split("\n")).toHaveLength(30);
});
test("shows an upstream timeout as a timeout", async ({ page }) => {
  await page.route("**/api/v1/songlists", (route) =>
    route.fulfill({
      status: 504,
      json: { code: -1, msg: "获取超时，请稍后重试", data: null },
    }),
  );
  await page.goto("/");
  await page.getByRole("button", { name: "填入范例链接", exact: true }).click();
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("获取超时");
  await expect(
    page.getByRole("button", { name: "复制结果", exact: true }),
  ).toBeDisabled();
});
test("cancels without stale data or an error and can retry", async ({
  page,
}) => {
  let calls = 0;
  let releaseFirst!: () => void;
  let firstStarted!: () => void;
  const firstResponse = new Promise<void>((resolve) => {
    releaseFirst = resolve;
  });
  const firstRequest = new Promise<void>((resolve) => {
    firstStarted = resolve;
  });
  await page.route("**/api/v1/songlists", async (route) => {
    calls++;
    if (calls === 1) {
      firstStarted();
      await firstResponse;
    }
    await route.fulfill({ json: playlist() }).catch(() => {});
  });
  await page.goto("/");
  await page
    .getByLabel("公开歌单链接或酷狗码")
    .fill("https://y.qq.com/n/ryqq/playlist/1");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await firstRequest;
  await page.getByRole("button", { name: "取消", exact: true }).click();
  releaseFirst();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByLabel("可编辑的歌曲文本")).toBeEnabled();
  expect(await page.getByLabel("可编辑的歌曲文本").inputValue()).toContain(
    "歌曲 1",
  );
});
test("persists the language preference", async ({ page }, testInfo) => {
  await page.route("https://img.shields.io/**", (route) =>
    route.fulfill({
      contentType: "image/svg+xml",
      body: '<svg xmlns="http://www.w3.org/2000/svg" width="120" height="20"><rect width="120" height="20" fill="#555"/></svg>',
    }),
  );
  await page.goto("/");
  const language = page.getByRole("button", { name: "Switch to English" });
  await expect(language).toHaveText("English");
  const size = await language.boundingBox();
  expect(size!.height).toBeGreaterThanOrEqual(34);
  await page.setViewportSize({ width: 320, height: 844 });
  await expect(page.getByAltText("GitHub stars")).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth > innerWidth,
    ),
  ).toBe(false);
  await page.screenshot({
    path: `test-results/${testInfo.project.name}-320.png`,
    fullPage: true,
  });
  await language.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("button", { name: "切换到中文" })).toHaveText(
    "中文",
  );
  await expect(
    page.getByRole("button", { name: "Get playlist", exact: true }),
  ).toBeVisible();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Export and transfer playlists",
  );
  const sources = page.getByRole("list", {
    name: "Supported public playlist sources",
  });
  await expect(sources).toBeInViewport({ ratio: 1 });
  await expect(sources.getByRole("listitem")).toHaveCount(9);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.reload();
  await expect(
    page.getByRole("button", { name: "Get playlist", exact: true }),
  ).toBeVisible();
});

test("uses the previous Shanghai Sunday in both languages and keeps the example", async ({
  page,
}) => {
  await page.clock.setFixedTime(new Date("2026-11-15T15:59:59.999Z"));
  await page.goto("/");
  await expect(
    page.getByText("赞赏榜 · 更新于 2026.11.08", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "填入范例链接", exact: true }).click();
  await expect(page.getByLabel("公开歌单链接或酷狗码")).toHaveValue(
    "http://163cn.tv/zoIxm3",
  );
  await page.getByRole("button", { name: "Switch to English" }).click();
  await expect(
    page.getByText("Top supporters · Updated 2026.11.08", { exact: true }),
  ).toBeVisible();
  await page.clock.setFixedTime(new Date("2026-11-15T16:00:00.000Z"));
  await page.reload();
  await expect(
    page.getByText("Top supporters · Updated 2026.11.15", { exact: true }),
  ).toBeVisible();
  await expect(page.locator("tbody tr").first()).toContainText("冰楠");
  await expect(page.locator("tbody tr").first()).toContainText("188");
});

test("shows Kugou's 10 of 12 preview as incomplete", async ({ page }) => {
  const result = playlist(10, false);
  await page.route("**/api/v1/songlists", (route) =>
    route.fulfill({
      json: {
        ...result,
        data: {
          ...result.data,
          source: "kugou",
          total_count: 12,
          preview: true,
        },
      },
    }),
  );
  await page.goto("/");
  await page
    .getByLabel("公开歌单链接或酷狗码")
    .fill(
      "https://m.kugou.com/songlist/gcid_3zbivkavz4z072/?src_cid=3zbivkavz4z072&uid=606105881&chl=wechat&iszlist=1",
    );
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByText("完整性未确认", { exact: true })).toBeVisible();
  await expect(page.getByText("平台显示 12", { exact: false })).toBeVisible();
  await expect(
    page.getByText("该酷狗分享页仅提供公开预览。", { exact: false }),
  ).toBeVisible();
  expect(
    (await page.getByLabel("可编辑的歌曲文本").inputValue()).split("\n"),
  ).toHaveLength(10);
});

test("replaces a Kugou preview with the complete twelve tracks on retry", async ({
  page,
}) => {
  let requests = 0;
  await page.route("**/api/v1/songlists", (route) => {
    const partial = requests++ === 0;
    const result = playlist(partial ? 10 : 12, !partial);
    return route.fulfill({
      json: {
        ...result,
        data: {
          ...result.data,
          source: "kugou",
          total_count: 12,
          ...(partial ? { preview: true } : {}),
        },
      },
    });
  });
  await page.goto("/");
  await page
    .getByLabel("公开歌单链接或酷狗码")
    .fill("https://m.kugou.com/songlist/gcid_3zbivkavz4z072/");
  const fetch = page.getByRole("button", { name: "获取歌单", exact: true });
  await fetch.click();
  await expect(page.getByText("完整性未确认", { exact: true })).toBeVisible();
  await fetch.click();
  await expect(page.getByText("完整性未确认", { exact: true })).toHaveCount(0);
  await expect(
    page.getByText("该酷狗分享页仅提供公开预览。", { exact: false }),
  ).toHaveCount(0);
  await expect(page.locator(".result-heading")).toContainText(
    "已获取 12 首歌曲",
  );
  expect(
    (await page.getByLabel("可编辑的歌曲文本").inputValue()).split("\n"),
  ).toHaveLength(12);
  expect(requests).toBe(2);
});

test("accepts raw and labelled Kugou codes and exports all twelve tracks", async ({
  page,
}) => {
  const inputs: string[] = [];
  await page.route("**/api/v1/songlists", (route) => {
    inputs.push(route.request().postDataJSON().url);
    const result = playlist(12);
    result.data.source = "kugou";
    return route.fulfill({ json: result });
  });
  await page.goto("/");
  const input = page.getByLabel("公开歌单链接或酷狗码");
  await expect(input).toHaveAttribute("placeholder", /酷狗码/);
  for (const code of ["40725013", "酷狗码：40725013"]) {
    await input.fill(code);
    await expect(page.getByText("已识别平台：酷狗音乐")).toBeVisible();
    await page.getByRole("button", { name: "获取歌单", exact: true }).click();
    await expect(page.locator(".result-heading")).toContainText(
      "已获取 12 首歌曲",
    );
    expect(
      (await page.getByLabel("可编辑的歌曲文本").inputValue()).split("\n"),
    ).toHaveLength(12);
  }
  expect(inputs).toEqual(["40725013", "酷狗码：40725013"]);
  const output = page.getByLabel("可编辑的歌曲文本");
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "下载 TXT", exact: true }).click();
  const stream = await (await download).createReadStream();
  const chunks: Buffer[] = [];
  for await (const chunk of stream!) chunks.push(Buffer.from(chunk));
  expect(
    Buffer.concat(chunks)
      .toString("utf8")
      .replace(/^\ufeff/, ""),
  ).toBe(await output.inputValue());
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("explains expired Kugou codes in both languages and permits retry", async ({
  page,
}) => {
  let calls = 0;
  await page.route("**/api/v1/songlists", (route) => {
    if (calls++ === 0)
      return route.fulfill({
        status: 400,
        json: { code: -1, msg: "酷狗码无效或已过期", data: null },
      });
    const result = playlist(12);
    result.data.source = "kugou";
    return route.fulfill({ json: result });
  });
  await page.goto("/");
  await page.getByLabel("公开歌单链接或酷狗码").fill("40725013");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("酷狗码无效、已过期");
  await expect(
    page.getByRole("button", { name: "复制结果", exact: true }),
  ).toBeDisabled();
  await page.getByRole("button", { name: "Switch to English" }).click();
  await expect(page.getByRole("alert")).toContainText(
    "Kugou code is invalid, expired",
  );
  await page.getByRole("button", { name: "Get playlist", exact: true }).click();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.locator(".result-heading")).toContainText("12 tracks");
  expect(calls).toBe(2);
});
