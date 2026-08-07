"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { runtimeListOptions } from "@multica/core/runtimes/queries";
import { runtimeModelsOptions } from "@multica/core/runtimes";
import { agentListOptions, squadListOptions } from "@multica/core/workspace/queries";
import { useCurrentWorkspace } from "@multica/core/paths";
import type { Agent, RuntimeModel, RuntimeDevice } from "@multica/core/types";

export interface RunModelOverrides {
  /** False when the issue has no runnable agent/squad assignee. */
  enabled: boolean;
  runtimeId: string | null;
  runtimeOnline: boolean;
  models: RuntimeModel[];
  /** Catalog entry matching the currently selected model (or the agent's
   *  persisted model when no override is set). */
  activeModel?: RuntimeModel;
  model: string;
  thinkingLevel: string;
  setModel: (next: string) => void;
  setThinkingLevel: (next: string) => void;
}

/**
 * Resolves the runtime behind an issue's assignee and exposes the same
 * per-run model / thinking-level override state used by quick-create. The
 * pickers stay hidden for unassigned or member-assigned issues.
 */
export function useRunModelOverrides(
  assigneeType?: string,
  assigneeId?: string,
  initialModel?: string,
  initialThinkingLevel?: string,
): RunModelOverrides {
  const ws = useCurrentWorkspace();
  const wsId = ws?.id ?? "";
  const { data: agents = [] } = useQuery({ ...agentListOptions(wsId), enabled: !!wsId });
  const { data: squads = [] } = useQuery({ ...squadListOptions(wsId), enabled: !!wsId });
  const { data: runtimes = [] } = useQuery({ ...runtimeListOptions(wsId), enabled: !!wsId });

  const agent = useMemo<Agent | undefined>(() => {
    if (!assigneeType || !assigneeId) return undefined;
    if (assigneeType === "agent") {
      return agents.find((a) => a.id === assigneeId && !a.archived_at);
    }
    if (assigneeType === "squad") {
      const squad = squads.find((s) => s.id === assigneeId && !s.archived_at);
      return agents.find((a) => a.id === squad?.leader_id && !a.archived_at);
    }
    return undefined;
  }, [agents, squads, assigneeType, assigneeId]);

  const runtimeId = agent?.runtime_id || null;
  const runtime = useMemo<RuntimeDevice | undefined>(
    () => (runtimeId ? runtimes.find((r) => r.id === runtimeId) : undefined),
    [runtimes, runtimeId],
  );
  const modelsQuery = useQuery(runtimeModelsOptions(runtimeId));
  const models = useMemo(
    () => modelsQuery.data?.models ?? [],
    [modelsQuery.data],
  );

  const [model, setModelState] = useState(initialModel ?? "");
  const [thinkingLevel, setThinkingLevel] = useState(initialThinkingLevel ?? "");
  useEffect(() => {
    setModelState("");
    setThinkingLevel("");
  }, [assigneeType, assigneeId, runtimeId]);

  const activeModel = useMemo(
    () =>
      models.find((m) => m.id === model) ??
      (!model ? models.find((m) => m.id === agent?.model) : undefined),
    [models, model, agent?.model],
  );

  return {
    enabled: !!agent && !!runtimeId,
    runtimeId,
    runtimeOnline: runtime?.status === "online",
    models,
    activeModel,
    model,
    thinkingLevel,
    setModel: (next) => {
      setModelState(next);
      setThinkingLevel("");
    },
    setThinkingLevel,
  };
}
