"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@multica/core/auth";

const BOARD_PATH = "/codex-board/issues";

export default function LandingPage() {
  const router = useRouter();

  useEffect(() => {
    let cancelled = false;
    async function bootstrap() {
      try {
        const me = await fetch("/api/me", { credentials: "include" });
        if (!cancelled && me.ok) {
          router.replace(BOARD_PATH);
          return;
        }

        const configResponse = await fetch("/api/config", {
          credentials: "include",
        });
        const config = configResponse.ok
          ? ((await configResponse.json()) as {
              auto_login_email?: string;
              auto_login_code?: string;
            })
          : {};
        const email =
          config.auto_login_email ??
          process.env.NEXT_PUBLIC_AUTO_LOGIN_EMAIL ??
          "local@multica.local";
        const code =
          config.auto_login_code ??
          process.env.NEXT_PUBLIC_AUTO_LOGIN_CODE ??
          "888888";

        await fetch("/auth/send-code", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email }),
        });
        const verifyResponse = await fetch("/auth/verify-code", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email, code }),
        });
        const verifyBody = await verifyResponse.json();
        if (verifyBody?.token) {
          await useAuthStore.getState().loginWithToken(verifyBody.token);
        }

        if (!cancelled) router.replace(BOARD_PATH);
      } catch {
        if (!cancelled) router.replace("/login");
      }
    }
    void bootstrap();
    return () => {
      cancelled = true;
    };
  }, [router]);

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        height: "100vh",
        fontFamily: "system-ui, sans-serif",
      }}
    >
      正在进入看板…
    </div>
  );
}
