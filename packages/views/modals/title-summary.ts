// Manual create no longer forces the user to write a title. When the title is
// empty, this derives a 10-20 character board title from the description:
// markdown is stripped, then whole sentences are kept until the lower bound is
// reached, falling back to a hard character truncation for long single
// sentences. Kept deliberately deterministic so the same description always
// yields the same board title.
export function summarizeTitle(markdown: string, min = 10, max = 20): string {
  const plain = toPlainText(markdown);
  if (!plain) return "未命名任务";

  const chars = Array.from(plain);
  if (chars.length <= max) return plain;

  const sentences =
    plain.match(/[^。！？!?；;]+[。！？!?；;]*/g)?.map((s) => s.trim()).filter(Boolean) ??
    [];
  let acc = "";
  for (const sentence of sentences) {
    const next = acc ? `${acc} ${sentence}` : sentence;
    if (Array.from(next).length > max) break;
    acc = next;
    if (Array.from(acc).length >= min) break;
  }
  if (acc) return acc;

  return chars.slice(0, max).join("");
}

function toPlainText(markdown: string): string {
  return markdown
    .replace(/!\[[^\]]*\]\([^)]*\)/g, " ")
    .replace(/\[\/[^\]]*\]\(slash:\/\/skill\/[^)]+\)/g, " ")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/[*_`~#>|]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}
