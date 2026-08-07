"use client";

import { Brain, Cpu } from "lucide-react";
import { cn } from "@multica/ui/lib/utils";
import { ModelPicker } from "../../agents/components/inspector/model-picker";
import { ThinkingPicker } from "../../agents/components/inspector/thinking-picker";
import type { RunModelOverrides } from "../hooks/use-run-model-overrides";

export function RunOverridePickers({
  overrides,
  className,
}: {
  overrides: RunModelOverrides;
  className?: string;
}) {
  if (!overrides.enabled) return null;
  const levels = overrides.activeModel?.thinking?.supported_levels ?? [];

  return (
    <div className={cn("flex items-center gap-1.5", className)}>
      <span className="flex items-center gap-1.5 rounded-md border border-border/70 bg-background/60 px-2 py-1 text-caption text-muted-foreground">
        <Cpu className="size-3.5 shrink-0" aria-hidden="true" />
        <span className="shrink-0">模型</span>
        <ModelPicker
          runtimeId={overrides.runtimeId}
          runtimeOnline={overrides.runtimeOnline}
          value={overrides.model}
          canEdit
          variant="chip"
          showLabel={false}
          onChange={overrides.setModel}
        />
      </span>
      <span className="flex items-center gap-1.5 rounded-md border border-border/70 bg-background/60 px-2 py-1 text-caption text-muted-foreground">
        <Brain className="size-3.5 shrink-0" aria-hidden="true" />
        <span className="shrink-0">推理</span>
        <ThinkingPicker
          value={overrides.thinkingLevel}
          levels={levels}
          canEdit={levels.length > 0}
          variant="chip"
          showLabel={false}
          onChange={overrides.setThinkingLevel}
        />
      </span>
    </div>
  );
}
