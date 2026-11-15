// Weeks run Monday–Sunday in Beijing time, independently of the visitor's zone.
export function lastCompletedWeekDate(now = new Date()): string {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Shanghai",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(now);
  const part = (type: Intl.DateTimeFormatPartTypes) =>
    Number(parts.find((value) => value.type === type)!.value);
  // Perform calendar arithmetic in UTC after selecting the Shanghai date.
  const end = new Date(Date.UTC(part("year"), part("month") - 1, part("day")));
  end.setUTCDate(end.getUTCDate() - (end.getUTCDay() || 7));
  return [
    end.getUTCFullYear(),
    String(end.getUTCMonth() + 1).padStart(2, "0"),
    String(end.getUTCDate()).padStart(2, "0"),
  ].join(".");
}
