package daemon

import (
	"strings"
	"testing"
)

func TestBuildPromptIncludesChineseOutputRule(t *testing.T) {
	out := BuildPrompt(Task{IssueID: "issue-1"}, "codex")
	for _, want := range []string{"## Output Language", "Simplified Chinese", "中文"} {
		if !strings.Contains(out, want) {
			t.Fatalf("prompt missing %q\n---\n%s", want, out)
		}
	}
}

func TestBuildPromptQuickCreateOmitsChineseOutputRule(t *testing.T) {
	out := BuildPrompt(Task{QuickCreatePrompt: "做一个登录页"}, "codex")
	if strings.Contains(out, "## Output Language") {
		t.Fatalf("quick-create prompt should keep its strict machine output format\n---\n%s", out)
	}
}
