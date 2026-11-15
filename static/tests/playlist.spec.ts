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
  await expect(page.getByRole("heading", { level: 1 })).toContainText(
    "迁移网易云",
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
    .getByLabel("歌单分享链接")
    .fill("https://y.qq.com.evil.test/playlist/1");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("官方歌单链接");
  await page.route("**/api/v1/songlists", (route) => route.abort("failed"));
  await page
    .getByLabel("歌单分享链接")
    .fill("https://y.qq.com/n/ryqq/playlist/1");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("连接失败");
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
    .getByLabel("歌单分享链接")
    .fill("分享歌单 https://y.qq.com/n/ryqq/playlist/1");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  const output = page.getByLabel("可编辑的歌单文本");
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
    .getByLabel("歌单分享链接")
    .fill("https://y.qq.com/n/ryqq/playlist/1");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByText("部分歌曲未获取", { exact: true })).toBeVisible();
  await expect(page.getByText("平台显示 531", { exact: false })).toBeVisible();
  await page.getByLabel("分批导出").selectOption("500");
  expect(
    (await page.getByLabel("可编辑的歌单文本").inputValue()).split("\n"),
  ).toHaveLength(500);
  await page.getByRole("button", { name: "下一批" }).click();
  expect(await page.getByLabel("可编辑的歌单文本").inputValue()).toBe(
    "歌曲 501 - Live - 歌手甲 / 歌手乙",
  );
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
    .getByLabel("歌单分享链接")
    .fill("https://y.qq.com/n/ryqq/playlist/1");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await firstRequest;
  await page.getByRole("button", { name: "取消", exact: true }).click();
  releaseFirst();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByLabel("可编辑的歌单文本")).toBeEnabled();
  expect(await page.getByLabel("可编辑的歌单文本").inputValue()).toContain(
    "歌曲 1",
  );
});
test("persists the language preference", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Switch to English" }).click();
  await expect(
    page.getByRole("button", { name: "Get playlist", exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(
    page.getByRole("button", { name: "Get playlist", exact: true }),
  ).toBeVisible();
});
