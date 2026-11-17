import { expect, test } from "@playwright/test";

function playlist(titles: string[]) {
  return {
    code: 1,
    msg: "success",
    data: {
      name: "编号测试歌单",
      source: "kugou",
      tracks: titles.map((title, i) => ({
        id: `${i + 1}`,
        title,
        artists: ["歌手"],
      })),
      songs: titles.map((title) => `${title} - 歌手`),
      songs_count: titles.length,
      total_count: titles.length,
      complete: true,
      warnings: [],
    },
  };
}

async function load(page: import("@playwright/test").Page, titles: string[]) {
  await page.route("**/api/v1/songlists", (route) =>
    route.fulfill({ json: playlist(titles) }),
  );
  await page.goto("/");
  await page.getByLabel("公开歌单链接或酷狗码").fill("40725013");
  await page.getByRole("button", { name: "获取歌单", exact: true }).click();
  await expect(page.getByLabel("可编辑的歌曲文本")).toBeEnabled();
}

test("line numbers stay out of edited text, selection, clipboard and TXT", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await load(page, ["Wildest Dreams", "Normal No More", "Escapism."]);
  const rows = page.locator(".editor-line");
  await expect(rows).toHaveCount(3);
  await expect(rows.nth(0)).toHaveAttribute("data-line", "1");
  await expect(rows.nth(2)).toHaveAttribute("data-line", "3");
  const output = page.getByLabel("可编辑的歌曲文本");
  const gutter = await page.evaluate(() => {
    const area = document.querySelector(".result-text")!;
    const number = getComputedStyle(
      document.querySelector(".editor-line")!,
      "::before",
    );
    const extra =
      number.boxSizing === "border-box"
        ? 0
        : [
            number.paddingLeft,
            number.paddingRight,
            number.borderLeftWidth,
            number.borderRightWidth,
          ].reduce((sum, width) => sum + parseFloat(width), 0);
    return {
      available: parseFloat(getComputedStyle(area).marginLeft),
      painted: parseFloat(number.width) + extra,
    };
  });
  expect(gutter.painted).toBeLessThanOrEqual(gutter.available + 0.5);
  const text = "Edited song - Artist\n\nAnother song - Singer\n";
  await output.fill(text);
  await expect(rows).toHaveCount(4);
  await expect(rows.nth(3)).toHaveAttribute("data-line", "4");
  await expect(page.locator(".editor-lines")).toHaveAttribute(
    "aria-hidden",
    "true",
  );
  expect(
    await page
      .locator(".editor-lines")
      .evaluate((el) => getComputedStyle(el).userSelect),
  ).toBe("none");
  await output.focus();
  await page.keyboard.press("ControlOrMeta+A");
  expect(
    await output.evaluate((el: HTMLTextAreaElement) =>
      el.value.slice(el.selectionStart, el.selectionEnd),
    ),
  ).toBe(text);
  await page.keyboard.press("ControlOrMeta+C");
  await expect
    .poll(() => page.evaluate(() => navigator.clipboard.readText()))
    .toBe(text);
  await page.getByRole("button", { name: "复制结果", exact: true }).click();
  await expect(page.getByText("已复制到剪贴板", { exact: true })).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(text);
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "下载 TXT", exact: true }).click();
  const stream = await (await download).createReadStream();
  const chunks: Buffer[] = [];
  for await (const chunk of stream!) chunks.push(Buffer.from(chunk));
  expect(Buffer.concat(chunks).toString("utf8")).toBe(`\ufeff${text}`);
  await output.fill("");
  await expect(rows).toHaveCount(1);
});

test("line numbers align with native wrapping after scroll and resize", async ({
  page,
}) => {
  await load(page, [
    "Long song title with multiple artists and versions ".repeat(14),
    "第二首",
    ...Array.from({ length: 30 }, (_, i) => `歌曲 ${i + 3}`),
  ]);
  const output = page.getByLabel("可编辑的歌曲文本");
  const alignment = () =>
    page.evaluate(() => {
      const area = document.querySelector<HTMLTextAreaElement>(".result-text")!;
      const rows = document.querySelectorAll<HTMLElement>(".editor-line");
      const style = getComputedStyle(area);
      const probe = document.createElement("textarea");
      for (const name of [
        "font",
        "line-height",
        "padding",
        "letter-spacing",
        "white-space",
        "overflow-wrap",
        "word-break",
        "tab-size",
        "box-sizing",
      ])
        probe.style.setProperty(name, style.getPropertyValue(name));
      probe.style.cssText += `;position:fixed;left:-10000px;width:${area.getBoundingClientRect().width - (area.offsetWidth - area.clientWidth)}px;height:0;min-height:0;border:0;overflow:hidden;`;
      probe.value = area.value.split("\n")[0];
      document.body.append(probe);
      const nativeFirstHeight =
        probe.scrollHeight -
        parseFloat(style.paddingTop) -
        parseFloat(style.paddingBottom);
      probe.remove();
      const measuredFirstHeight =
        rows[1].getBoundingClientRect().top -
        rows[0].getBoundingClientRect().top;
      const expectedTop =
        area.getBoundingClientRect().top +
        parseFloat(style.paddingTop) -
        area.scrollTop;
      return {
        nativeFirstHeight,
        measuredFirstHeight,
        topError: Math.abs(rows[0].getBoundingClientRect().top - expectedTop),
        scrollTop: area.scrollTop,
      };
    });
  await expect
    .poll(async () =>
      Math.abs(
        (await alignment()).nativeFirstHeight -
          (await alignment()).measuredFirstHeight,
      ),
    )
    .toBeLessThanOrEqual(1);
  await output.evaluate((el) => {
    el.scrollTop = 170;
  });
  await expect
    .poll(async () => (await alignment()).topError)
    .toBeLessThanOrEqual(1);
  await output.evaluate((el) => {
    el.style.height = "360px";
  });
  await page.setViewportSize({ width: 320, height: 844 });
  await expect
    .poll(async () =>
      Math.abs(
        (await alignment()).nativeFirstHeight -
          (await alignment()).measuredFirstHeight,
      ),
    )
    .toBeLessThanOrEqual(1);
  await expect
    .poll(async () => (await alignment()).topError)
    .toBeLessThanOrEqual(1);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth > innerWidth,
    ),
  ).toBe(false);
});

test("line numbers continue across batches and follow inserted lines", async ({
  page,
}) => {
  await load(
    page,
    Array.from({ length: 501 }, (_, i) => `Song ${i + 1}`),
  );
  await page.getByRole("radio", { name: "每批 500 首", exact: true }).check();
  await expect(page.locator(".editor-line")).toHaveCount(500);
  await page.getByRole("button", { name: "下一批", exact: true }).click();
  await expect(page.locator(".editor-line")).toHaveCount(1);
  await expect(page.locator(".editor-line").first()).toHaveAttribute(
    "data-line",
    "501",
  );
  const output = page.getByLabel("可编辑的歌曲文本");
  await output.fill("Song 501 - Singer\nNew song - Singer");
  await expect(page.locator(".editor-line").last()).toHaveAttribute(
    "data-line",
    "502",
  );
  await page.getByRole("radio", { name: "全部歌曲", exact: true }).check();
  await expect(page.locator(".editor-line").last()).toHaveAttribute(
    "data-line",
    "501",
  );
  await page.getByRole("radio", { name: "倒序", exact: true }).check();
  await expect(page.locator(".editor-line").first()).toHaveAttribute(
    "data-line",
    "1",
  );
  expect(await output.inputValue()).toMatch(/^Song 501/);
});

test("a ten-thousand-track editor keeps the last number visible when scrolled", async ({
  page,
}) => {
  await load(
    page,
    Array.from({ length: 10000 }, (_, i) => `Song ${i + 1}`),
  );
  const output = page.getByLabel("可编辑的歌曲文本");
  await expect(page.locator(".editor-line")).toHaveCount(10000);
  await expect(page.locator(".editor-line").last()).toHaveAttribute(
    "data-line",
    "10000",
  );
  await output.evaluate((el) => {
    el.scrollTop = el.scrollHeight;
  });
  await expect
    .poll(() =>
      page.evaluate(() => {
        const area = document
          .querySelector(".result-text")!
          .getBoundingClientRect();
        const last = document
          .querySelector(".editor-line:last-child")!
          .getBoundingClientRect();
        return last.top >= area.top && last.bottom <= area.bottom;
      }),
    )
    .toBe(true);
});
