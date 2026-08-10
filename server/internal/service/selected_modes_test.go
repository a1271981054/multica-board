package service

import "testing"

func TestSelectedModesFromTextNormalizesAlias(t *testing.T) {
	modes := SelectedModesFromText("[/目标模式](slash://skill/mode:目标模式)")
	if len(modes) != 1 || modes[0] != "目标" {
		t.Fatalf("modes = %v, want [目标]", modes)
	}
}

func TestSelectedModesFromTextPlainPhrase(t *testing.T) {
	modes := SelectedModesFromText("请用目标模式处理")
	if len(modes) != 1 || modes[0] != "目标" {
		t.Fatalf("modes = %v, want [目标]", modes)
	}
}

func TestSelectedModesFromMetadataRoundTrip(t *testing.T) {
	value, err := MarshalSelectedModes([]string{"目标模式", "计划模式", "目标"})
	if err != nil {
		t.Fatalf("MarshalSelectedModes: %v", err)
	}
	metadata := []byte(`{"selected_modes":` + string(value) + `}`)
	modes := SelectedModesFromMetadata(metadata)
	if len(modes) != 2 || modes[0] != "目标" || modes[1] != "计划模式" {
		t.Fatalf("modes = %v, want [目标 计划模式]", modes)
	}
}

func TestSelectedModesFromMetadataEmpty(t *testing.T) {
	if modes := SelectedModesFromMetadata([]byte(`{}`)); modes != nil {
		t.Fatalf("modes = %v, want nil", modes)
	}
}
