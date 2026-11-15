import { chromium } from "playwright";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const root = process.env.GOMUSIC_WEB_URL || "http://127.0.0.1:8081";
const evidence = process.env.GOMUSIC_UI_EVIDENCE_DIR || "test-results/live";
const link = process.env.GOMUSIC_UI_PLAYLIST_URL || "http://163cn.tv/zoIxm3";
await mkdir(evidence, { recursive: true });
const browser = await chromium.launch({
  channel: process.env.CI ? undefined : "chrome",
  headless: true,
});
const results = [];
try {
  for (const [name, viewport] of [
    ["desktop", { width: 1440, height: 1000 }],
    ["mobile", { width: 390, height: 844 }],
  ]) {
    const context = await browser.newContext({
      viewport,
      deviceScaleFactor: 1,
      permissions: ["clipboard-read", "clipboard-write"],
    });
    const page = await context.newPage();
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.goto(root, { waitUntil: "domcontentloaded" });
    await page.getByAltText("WeChat").scrollIntoViewIfNeeded();
    await page
      .getByAltText("WeChat")
      .evaluate((image) => image.decode().catch(() => {}));
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      path: path.join(evidence, `${name}-initial.png`),
      fullPage: true,
    });
    if ((await page.getByLabel("歌单链接或酷狗码").inputValue()) !== "")
      throw new Error("Initial playlist input must be empty");
    await page.getByLabel("歌单链接或酷狗码").fill(link);
    const response = page.waitForResponse((response) =>
      response.url().endsWith("/api/v1/songlists"),
    );
    await page.getByRole("button", { name: "获取歌单", exact: true }).click();
    const api = await response;
    const body = await api.json();
    if (!api.ok() || body.code !== 1)
      throw new Error(`Live request failed: ${api.status()} ${body.msg}`);
    await page.getByLabel("可编辑的歌单文本").waitFor({ timeout: 100_000 });
    await page.getByRole("button", { name: "复制结果", exact: true }).click();
    await page.getByText("已复制到剪贴板", { exact: true }).waitFor();
    const copied = await page.evaluate(() => navigator.clipboard.readText());
    if (copied !== (await page.getByLabel("可编辑的歌单文本").inputValue()))
      throw new Error("Clipboard does not match export");
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      path: path.join(evidence, `${name}-result.png`),
      fullPage: true,
    });
    await page.screenshot({ path: path.join(evidence, `${name}-main.png`) });
    const layout = await page.evaluate(() => ({
      overflow: document.documentElement.scrollWidth > innerWidth,
      bodyFont: getComputedStyle(document.querySelector("textarea")).fontSize,
      inputTop: document.querySelector("#playlist-link").getBoundingClientRect()
        .top,
    }));
    await page.getByRole("button", { name: "Switch to English" }).click();
    await page.screenshot({
      path: path.join(evidence, `${name}-english.png`),
      fullPage: true,
    });
    results.push({
      name,
      layout,
      errors,
      source: body.data.source,
      title: body.data.name,
      fetched: body.data.songs_count,
      total: body.data.total_count,
      complete: body.data.complete,
      clipboardMatches: true,
    });
    await context.close();
  }
  await writeFile(
    path.join(evidence, "inspection.json"),
    JSON.stringify(results, null, 2),
  );
  console.log(JSON.stringify(results, null, 2));
} finally {
  await browser.close();
}
