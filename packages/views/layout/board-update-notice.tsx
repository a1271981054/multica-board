"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight, X } from "lucide-react";
import { api } from "@multica/core/api";
import { useNavigation } from "../navigation";
import { useWorkspacePaths } from "@multica/core/paths";
import { displayBoardVersion } from "../common/board-version";

export function BoardUpdateNotice() {
  const [dismissed, setDismissed] = useState(false);
  const navigation = useNavigation();
  const paths = useWorkspacePaths();
  const versionQuery = useQuery({
    queryKey: ["board", "version", "notice"],
    queryFn: () => api.getBoardVersion(),
    retry: false,
    staleTime: 5 * 60_000,
  });

  if (dismissed || versionQuery.isError || !versionQuery.data?.update_available) {
    return null;
  }

  return (
    <div className="flex items-center gap-3 border-b border-border bg-brand/5 px-4 py-2 text-caption">
      <span className="min-w-0 flex-1 truncate text-foreground">
        发现新版本 {displayBoardVersion(versionQuery.data.latest)}（当前 {displayBoardVersion(versionQuery.data.current)}）
      </span>
      <button
        type="button"
        onClick={() => navigation.push(`${paths.settings()}?tab=system`)}
        className="inline-flex shrink-0 items-center gap-1 text-primary hover:underline"
      >
        去更新
        <ArrowRight className="size-3.5" />
      </button>
      <button
        type="button"
        onClick={() => setDismissed(true)}
        aria-label="关闭更新提示"
        className="shrink-0 rounded-sm p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
      >
        <X className="size-3.5" />
      </button>
    </div>
  );
}
