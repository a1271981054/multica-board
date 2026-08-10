package daemon

import (
	"regexp"
	"strings"
)

var slashToolRe = regexp.MustCompile(`\[/?([^\]\n]+)\]\(slash://skill/([^)\s]+)\)`)
var slashToolAnyRe = regexp.MustCompile(`\[/?[^\]\n]*\]\(slash://skill/[^)\s]+\)`)

// normalizeModeName maps user-facing aliases onto the canonical mode names
// activeModesBrief understands. The board now labels the goal picker
// `目标模式`; older tags and plain-text mentions still use `目标`.
func normalizeModeName(name string) string {
	if name == "目标模式" {
		return "目标"
	}
	return name
}

func selectedToolsFromText(text string) (modes []string, skills []string) {
	seenMode := map[string]struct{}{}
	for _, m := range slashToolRe.FindAllStringSubmatch(text, -1) {
		id := m[2]
		if strings.HasPrefix(id, "mode:") {
			mode := normalizeModeName(strings.TrimPrefix(id, "mode:"))
			if mode == "" {
				continue
			}
			if _, ok := seenMode[mode]; ok {
				continue
			}
			seenMode[mode] = struct{}{}
			modes = append(modes, mode)
		} else if id != "" {
			skills = append(skills, id)
		}
	}
	if strings.Contains(text, "目标模式") {
		if _, ok := seenMode["目标"]; !ok {
			seenMode["目标"] = struct{}{}
			modes = append(modes, "目标")
		}
	}
	return modes, skills
}

func collectSelectedTools(task Task) (modes []string, skills []string) {
	texts := []string{
		task.QuickCreatePrompt,
		task.TriggerCommentContent,
		task.ChatMessage,
		task.HandoffNote,
	}
	for _, c := range task.CoalescedComments {
		texts = append(texts, c.Content)
	}
	seenMode := map[string]struct{}{}
	seenSkill := map[string]struct{}{}
	for _, mode := range task.IssueSelectedModes {
		mode = normalizeModeName(mode)
		if mode == "" {
			continue
		}
		if _, ok := seenMode[mode]; ok {
			continue
		}
		seenMode[mode] = struct{}{}
		modes = append(modes, mode)
	}
	for _, text := range texts {
		ms, ss := selectedToolsFromText(text)
		for _, m := range ms {
			m = normalizeModeName(m)
			if _, ok := seenMode[m]; !ok {
				seenMode[m] = struct{}{}
				modes = append(modes, m)
			}
		}
		for _, s := range ss {
			if _, ok := seenSkill[s]; !ok {
				seenSkill[s] = struct{}{}
				skills = append(skills, s)
			}
		}
	}
	return modes, skills
}

func skillRefsForSelected(ids []string) []SkillRefData {
	refs := make([]SkillRefData, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, SkillRefData{
			ID:     id,
			Source: "workspace",
			Name:   id,
			Hash:   "pending",
		})
	}
	return refs
}

func activeModesBrief(modes []string) string {
	if len(modes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Active Modes\n")
	b.WriteString("The user selected the following modes for this task. Activate them explicitly and honor their behavior throughout the run:\n")
	for _, mode := range modes {
		switch mode {
		case "目标":
			b.WriteString("- **目标**: Treat this task as an explicit goal. Keep the objective visible, work toward completion, and only finish when the goal is achieved.\n")
		case "计划模式":
			b.WriteString("- **计划模式**: Use Plan mode. Produce a concrete plan and use the plan tool before making changes; do not start editing until the plan is established.\n")
		case "推理":
			b.WriteString("- **推理**: Use high reasoning effort for this task and show your analysis before conclusions.\n")
		case "模型":
			b.WriteString("- **模型**: Follow the selected model override for this task.\n")
		case "MCP":
			b.WriteString("- **MCP**: Prefer MCP-connected tools where they apply to the task.\n")
		case "状态":
			b.WriteString("- **状态**: Keep issue/task status updated and report progress explicitly.\n")
		case "反馈":
			b.WriteString("- **反馈**: End with a concise feedback summary of what changed and what remains.\n")
		case "侧边":
			b.WriteString("- **侧边**: Keep the conversation concise, sidebar-friendly, and scannable.\n")
		case "宠物":
			b.WriteString("- **宠物**: Treat this as a small companion-style task; be friendly and lightweight.\n")
		default:
			b.WriteString("- **" + mode + "**: Activate this mode and follow its behavior.\n")
		}
	}
	b.WriteString("\n")
	return b.String()
}

// taskGoalObjective picks the user-facing request to seed as a native Codex
// goal. Prefer the trigger/chat/quick-create text so the goal objective is the
// actual instruction; fall back to the issue title/body for assignment runs.
func taskGoalObjective(task Task) string {
	for _, text := range []string{
		task.TriggerCommentContent,
		task.ChatMessage,
		task.QuickCreatePrompt,
		task.HandoffNote,
	} {
		if trimmed := cleanGoalObjective(text); trimmed != "" {
			return trimmed
		}
	}
	if trimmed := cleanGoalObjective(task.IssueTitle); trimmed != "" {
		return trimmed
	}
	if trimmed := cleanGoalObjective(task.IssueDescription); trimmed != "" {
		return trimmed
	}
	return ""
}

// cleanGoalObjective strips slash-tool tags (mode/skill routing metadata) from
// the text used as a native Codex goal objective so the visible objective reads
// like the user's actual request instead of the routing tag.
func cleanGoalObjective(text string) string {
	return strings.TrimSpace(slashToolAnyRe.ReplaceAllString(text, ""))
}
