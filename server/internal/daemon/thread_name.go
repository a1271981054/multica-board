package daemon

import (
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/execenv"
)

const codexThreadNameMaxRunes = 120

func deriveTaskThreadName(task Task) string {
	candidates := []string{
		task.ThreadName,
		task.AutopilotTitle,
		task.QuickCreatePrompt,
		task.ChatMessage,
		task.TriggerCommentContent,
	}
	for _, candidate := range candidates {
		if name := normalizeThreadName(candidate, codexThreadNameMaxRunes); name != "" {
			return name
		}
	}
	return ""
}

func normalizeThreadName(s string, maxRunes int) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	normalized := strings.Join(fields, " ")
	if maxRunes <= 0 {
		return normalized
	}
	rs := []rune(normalized)
	if len(rs) <= maxRunes {
		return normalized
	}
	if maxRunes <= 3 {
		return string(rs[:maxRunes])
	}
	return string(rs[:maxRunes-3]) + "..."
}

// codexThreadSourceForEnv classifies shared-home sessions as ordinary user
// conversations so the Codex sidebar can show them in the same thread list
// instead of treating them as anonymous CLI sessions.
func codexThreadSourceForEnv(env *execenv.Environment) string {
	if env != nil && env.SharedCodexHome {
		return "user"
	}
	return ""
}
