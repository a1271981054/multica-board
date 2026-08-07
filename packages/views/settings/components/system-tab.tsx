"use client";

import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertCircle, CheckCircle2, Download, Loader2, RefreshCw } from "lucide-react";
import { api } from "@multica/core/api";
import { Button } from "@multica/ui/components/ui/button";
import { cn } from "@multica/ui/lib/utils";
import { SettingsCard, SettingsRow, SettingsSection, SettingsTab } from "./settings-layout";
import { displayBoardVersion } from "../../common/board-version";

export function SystemTab() {
  const versionQuery = useQuery({
    queryKey: ["board", "version"],
    queryFn: () => api.getBoardVersion(),
    retry: false,
    staleTime: 60_000,
  });
  const [phase, setPhase] = useState<"idle" | "starting" | "running" | "done" | "error">("idle");
  const [message, setMessage] = useState("");
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const stopPolling = () => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
  };
  useEffect(() => stopPolling, []);

  const startUpdate = async () => {
    setPhase("starting");
    setMessage("");
    try {
      const result = await api.startBoardUpdate();
      if (!result.started) {
        setPhase("error");
        setMessage(result.message || "当前环境不支持自动更新");
        return;
      }
      setPhase("running");
      timerRef.current = setInterval(async () => {
        try {
          const status = await api.getBoardUpdateStatus();
          setMessage(status.message ?? "");
          if (status.status === "done") {
            stopPolling();
            setPhase("done");
            void versionQuery.refetch();
          } else if (status.status === "error") {
            stopPolling();
            setPhase("error");
          }
        } catch {
          // The updater may have stopped the backend mid-apply; keep polling.
        }
      }, 2000);
    } catch {
      stopPolling();
      setPhase("error");
      setMessage("启动更新失败，请稍后重试。");
    }
  };

  const version = versionQuery.data;
  const loading = versionQuery.isLoading;
  const unavailable = versionQuery.isError || version?.message;
  const hasUpdate = version?.update_available === true;

  return (
    <SettingsTab title="系统升级">
      <SettingsSection
        title="版本信息"
        description="每次打开看板时会自动检查 GitHub 上的最新版本。"
      >
        <SettingsCard>
          <SettingsRow label="当前版本" description="本机安装的 Multica Board 版本">
            <span className="font-mono text-body text-foreground">
              {loading ? "检查中..." : displayBoardVersion(version?.current)}
            </span>
          </SettingsRow>
          <SettingsRow
            label="最新版本"
            description={hasUpdate ? "发现新版本，可以立即下载并更新。" : "当前已是最新版本。"}
          >
            <span className="font-mono text-body text-foreground">
              {loading ? "检查中..." : version?.latest ? displayBoardVersion(version.latest) : "未知"}
            </span>
          </SettingsRow>
        </SettingsCard>
      </SettingsSection>

      <SettingsSection title="更新操作" description="更新过程会自动停止服务、替换文件并重新启动。">
        <SettingsCard>
          <SettingsRow
            label={phase === "running" || phase === "starting" ? "正在更新" : "立即更新"}
            description={message || (phase === "running" ? "正在下载并应用新版本..." : "点击后将在后台下载最新版本并自动安装。")}
          >
            <div className="flex items-center gap-2">
              {phase === "running" || phase === "starting" ? (
                <span className="inline-flex items-center gap-1.5 text-caption text-muted-foreground">
                  <Loader2 className="size-3.5 animate-spin" />
                  更新中...
                </span>
              ) : phase === "done" ? (
                <span className="inline-flex items-center gap-1.5 text-caption text-success">
                  <CheckCircle2 className="size-3.5" />
                  更新完成
                </span>
              ) : phase === "error" ? (
                <span className="inline-flex items-center gap-1.5 text-caption text-destructive">
                  <AlertCircle className="size-3.5" />
                  更新失败
                </span>
              ) : null}
              <Button
                size="sm"
                onClick={() => void startUpdate()}
                disabled={phase === "running" || phase === "starting" || Boolean(unavailable) || !hasUpdate || loading}
              >
                {hasUpdate ? (
                  <>
                    <Download className="size-3.5" />
                    立即更新
                  </>
                ) : (
                  <>
                    <RefreshCw className="size-3.5" />
                    重新检查
                  </>
                )}
              </Button>
            </div>
          </SettingsRow>
        </SettingsCard>
      </SettingsSection>

      {unavailable && (
        <p className={cn("text-caption text-muted-foreground")}>
          当前环境没有启用自动更新（{version?.message ?? "接口不可用"}）。
        </p>
      )}
    </SettingsTab>
  );
}
