import { expect, it } from "vitest";
import { platformOf } from "./playlist";

it.each([
  "40725013",
  "  40725013\n",
  "酷狗码：40725013",
  "酷狗码 40725013",
  "Kugou code: 40725013",
  "00725013",
])("recognizes Kugou code input %s", (input) => {
  expect(platformOf(input)).toBe("kugou");
});

it.each([
  "40 725013",
  "40725013.0",
  "+40725013",
  "123456789012345678901",
  "酷狗码：40725013abc",
  "https://evil.test/?code=40725013",
])("does not mistake malformed input for a Kugou code: %s", (input) => {
  expect(platformOf(input)).toBeUndefined();
});
