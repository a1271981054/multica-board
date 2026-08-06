package agent

// Codex session-mode keys shared by the server API and the daemon. The value
// is stored inside the agent's custom_env (never in the secret-bearing env
// endpoints' audit surface) so each agent can opt between shared sidebar
// sessions and isolated parallel execution.
const (
	CodexSessionModeEnvKey   = "CODEX_SESSION_MODE"
	CodexSessionModeShared   = "shared"
	CodexSessionModeIsolated = "isolated"
)
