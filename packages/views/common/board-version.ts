export function displayBoardVersion(value?: string): string {
  if (!value) return "dev";
  const clean = value.replace(/^multica-board-v/, "").replace(/^v/, "");
  return clean === "dev" || clean === "" ? "dev" : `v${clean}`;
}
