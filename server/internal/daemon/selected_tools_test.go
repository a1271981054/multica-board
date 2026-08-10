package daemon

import (
	"strings"
	"testing"
)

func TestSelectedToolsFromText(t *testing.T) {
	modes, skills := selectedToolsFromText(
		"请用 [/目标](slash://skill/mode:目标) 和 [/deploy](slash://skill/skill-1) 处理",
	)
	if len(modes) != 1 || modes[0] != "目标" {
		t.Fatalf("modes = %v, want [目标]", modes)
	}
	if len(skills) != 1 || skills[0] != "skill-1" {
		t.Fatalf("skills = %v, want [skill-1]", skills)
	}
}

func TestSelectedToolsFromTextNormalizesGoalModeAlias(t *testing.T) {
	modes, _ := selectedToolsFromText(
		"请用 [/目标模式](slash://skill/mode:目标模式) 处理",
	)
	if len(modes) != 1 || modes[0] != "目标" {
		t.Fatalf("modes = %v, want [目标]", modes)
	}
}

func TestSelectedToolsFromTextPlainGoalModePhrase(t *testing.T) {
	modes, _ := selectedToolsFromText("请用目标模式处理这个任务")
	if len(modes) != 1 || modes[0] != "目标" {
		t.Fatalf("modes = %v, want [目标]", modes)
	}
}

func TestCollectSelectedToolsDeduplicates(t *testing.T) {
	task := Task{
		QuickCreatePrompt:     "[/目标](slash://skill/mode:目标) [/计划模式](slash://skill/mode:计划模式)",
		TriggerCommentContent: "[/deploy](slash://skill/skill-1) [/deploy](slash://skill/skill-1)",
		IssueSelectedModes:    []string{"目标模式", "目标"},
	}
	modes, skills := collectSelectedTools(task)
	if len(modes) != 2 {
		t.Fatalf("modes = %v, want 2 unique modes", modes)
	}
	if len(skills) != 1 || skills[0] != "skill-1" {
		t.Fatalf("skills = %v, want [skill-1]", skills)
	}
}

func TestTaskGoalObjectivePrefersTriggerOverIssue(t *testing.T) {
	task := Task{
		TriggerCommentContent: "[/目标](slash://skill/mode:目标) 改这里",
		IssueTitle:            "旧标题",
		IssueDescription:      "旧描述",
	}
	if got := taskGoalObjective(task); got != "改这里" {
		t.Fatalf("taskGoalObjective = %q, want cleaned trigger text", got)
	}
}

func TestTaskGoalObjectiveFallsBackToIssue(t *testing.T) {
	task := Task{
		IssueTitle:       "修复套餐编辑",
		IssueDescription: "门店套餐显示成公司套餐",
	}
	if got := taskGoalObjective(task); got != "修复套餐编辑" {
		t.Fatalf("taskGoalObjective = %q, want issue title", got)
	}
}

func TestTaskGoalObjectiveTagOnlyFallsBack(t *testing.T) {
	task := Task{
		TriggerCommentContent: "[/目标模式](slash://skill/mode:目标模式)",
		IssueTitle:            "修复套餐编辑",
	}
	if got := taskGoalObjective(task); got != "修复套餐编辑" {
		t.Fatalf("taskGoalObjective = %q, want issue title fallback", got)
	}
}

func TestActiveModesBrief(t *testing.T) {
	brief := activeModesBrief([]string{"目标", "计划模式"})
	for _, want := range []string{"## Active Modes", "Treat this task as an explicit goal", "Use Plan mode"} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief missing %q:\n%s", want, brief)
		}
	}
	if activeModesBrief(nil) != "" {
		t.Fatal("expected empty brief for no modes")
	}
}
