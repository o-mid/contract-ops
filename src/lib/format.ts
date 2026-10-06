export function formatTimestamp(value: string, style: "short" | "medium" = "short") {
  const dateStyle = style === "medium" ? "medium" : undefined;
  const timeStyle = style === "medium" ? "medium" : "short";
  return new Intl.DateTimeFormat("en-GB", {
    dateStyle,
    timeStyle,
    timeZone: "UTC"
  }).format(new Date(value));
}

export function formatLastUpdated(value: Date) {
  return new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "medium",
    timeZone: "UTC"
  }).format(value);
}
