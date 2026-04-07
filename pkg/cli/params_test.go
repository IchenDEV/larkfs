package cli

import (
	"encoding/json"
	"testing"
)

func TestJSONParam(t *testing.T) {
	result := JSONParam(map[string]any{
		"folder_token": `evil"value`,
		"count":        42,
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("JSONParam produced invalid JSON: %v", err)
	}

	if parsed["folder_token"] != `evil"value` {
		t.Errorf("expected evil\"value, got %v", parsed["folder_token"])
	}
}

func TestJSONParamSpecialChars(t *testing.T) {
	result := JSONParam(map[string]any{
		"token": `"}, "injected": "attack`,
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("JSONParam should safely escape special chars: %v", err)
	}

	if len(parsed) != 1 {
		t.Errorf("injection should not create extra keys, got %d keys", len(parsed))
	}
}
