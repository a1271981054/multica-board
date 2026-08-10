package service

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/multica-ai/multica/server/internal/util"
)

// SelectedModesMetadataKey is the reserved issue metadata key used to persist
// slash-selected modes from a quick-create run onto the issue it creates.
const SelectedModesMetadataKey = "selected_modes"

var slashModeRe = regexp.MustCompile(`\[/?([^\]\n]+)\]\(slash://skill/mode:([^)\s]+)\)`)

// NormalizeSelectedMode maps user-facing mode aliases onto the canonical mode
// names the daemon brief understands. Older board builds emitted `mode:目标`;
// the current UI lets users pick `目标模式` and we normalize both to `目标`.
func NormalizeSelectedMode(name string) string {
	if name == "目标模式" {
		return "目标"
	}
	return name
}

// SelectedModesFromText extracts slash-selected mode names from a raw text
// payload. It recognizes both the editor tag form
// (`[/目标模式](slash://skill/mode:目标模式)`) and the plain phrase
// `目标模式`, so typing the mode name without the slash menu still works.
func SelectedModesFromText(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var modes []string
	add := func(raw string) {
		mode := NormalizeSelectedMode(raw)
		if mode == "" {
			return
		}
		if _, ok := seen[mode]; ok {
			return
		}
		seen[mode] = struct{}{}
		modes = append(modes, mode)
	}
	for _, m := range slashModeRe.FindAllStringSubmatch(text, -1) {
		if len(m) > 2 {
			add(m[2])
		}
	}
	if strings.Contains(text, "目标模式") {
		add("目标")
	}
	return modes
}

// SelectedModesFromMetadata decodes the comma-separated mode list persisted
// under SelectedModesMetadataKey.
func SelectedModesFromMetadata(raw []byte) []string {
	meta := util.JSONObjectOrEmpty(raw)
	value, ok := meta[SelectedModesMetadataKey].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}
	var modes []string
	for _, part := range strings.Split(value, ",") {
		mode := NormalizeSelectedMode(strings.TrimSpace(part))
		if mode != "" {
			modes = append(modes, mode)
		}
	}
	return dedupeSelectedModes(modes)
}

// MarshalSelectedModes encodes modes for the issue metadata JSONB bag.
func MarshalSelectedModes(modes []string) ([]byte, error) {
	return json.Marshal(strings.Join(dedupeSelectedModes(modes), ","))
}

func dedupeSelectedModes(modes []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(modes))
	for _, mode := range modes {
		mode = NormalizeSelectedMode(mode)
		if mode == "" {
			continue
		}
		if _, ok := seen[mode]; ok {
			continue
		}
		seen[mode] = struct{}{}
		out = append(out, mode)
	}
	return out
}
