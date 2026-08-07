package daemon

import (
	"regexp"
	"strings"
)

var slashToolRe = regexp.MustCompile(`\[/?([^\]\n]+)\]\(slash://skill/([^)\s]+)\)`)

func selectedToolsFromText(text string) (modes []string, skills []string) {
	for _, m := range slashToolRe.FindAllStringSubmatch(text, -1) {
		id := m[2]
		if strings.HasPrefix(id, "mode:") {
			modes = append(modes, strings.TrimPrefix(id, "mode:"))
		} else if id != "" {
			skills = append(skills, id)
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
	for _, text := range texts {
		ms, ss := selectedToolsFromText(text)
		for _, m := range ms {
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
