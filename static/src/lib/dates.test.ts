import { afterEach, expect, it, vi } from "vitest";
import { lastCompletedWeekDate } from "./dates";

afterEach(() => vi.useRealTimers());

it.each([
  ["2026-11-15T15:59:59.999Z", "2026.11.08"],
  ["2026-11-15T16:00:00.000Z", "2026.11.15"],
  ["2026-11-21T15:00:00.000Z", "2026.11.15"],
  ["2026-11-22T04:00:00.000Z", "2026.11.15"],
  ["2023-12-31T16:00:00.000Z", "2023.12.31"],
  ["2024-03-01T00:00:00.000Z", "2024.02.25"],
  ["2026-02-28T16:00:00.000Z", "2026.02.22"],
])("finds the completed week for %s", (now, expected) => {
  expect(lastCompletedWeekDate(new Date(now))).toBe(expected);
});

it("uses the current date when the page starts", () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-11-15T16:00:00.000Z"));
  expect(lastCompletedWeekDate()).toBe("2026.11.15");
});
