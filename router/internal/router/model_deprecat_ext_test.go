package router

import (
	"context"
	"testing"

	"github.com/flowork-os/flowork_Router/internal/store"
)

func TestRemapDeprecatedModel(t *testing.T) {
	cases := map[string]struct {
		want     string
		remapped bool
	}{
		"claude-2.1":                 {"claude-opus-4-8", true},
		"claude-3-opus-20240229":     {"claude-opus-4-8", true},
		"anthropic/claude-3.5-sonnet": {"claude-sonnet-5", true},
		"gpt-3.5-turbo":              {"gpt-4o-mini", true},
		// HIDUP — jangan disentuh.
		"claude-haiku-4-5": {"claude-haiku-4-5", false},
		"claude-opus-4-8":  {"claude-opus-4-8", false},
		"claude-sonnet-5":  {"claude-sonnet-5", false},
		"model-random-xyz": {"model-random-xyz", false},
	}
	for in, exp := range cases {
		got, remapped := remapDeprecatedModel(in)
		if remapped != exp.remapped {
			t.Errorf("%q: remapped=%v mau %v", in, remapped, exp.remapped)
		}
		if got != exp.want {
			t.Errorf("%q: got %q mau %q", in, got, exp.want)
		}
	}
}

func TestModelRemap_ViaShaper(t *testing.T) {
	t.Setenv("FLOWORK_MODEL_REMAP", "1")
	out := applyInjectShaper(context.Background(),
		OpenAIRequest{Model: "claude-2.1", Messages: nil}, nil)
	if out.Model != "claude-opus-4-8" {
		t.Errorf("shaper harus remap deprecated: got %q", out.Model)
	}

	// Escape-hatch OFF → mentah (biar bisa test/expose provider lama sengaja).
	t.Setenv("FLOWORK_MODEL_REMAP", "0")
	out2 := applyInjectShaper(context.Background(),
		OpenAIRequest{Model: "claude-2.1"}, nil)
	if out2.Model != "claude-2.1" {
		t.Errorf("REMAP=0 harus biarin mentah: got %q", out2.Model)
	}

	// Model hidup ga boleh berubah walau ON.
	t.Setenv("FLOWORK_MODEL_REMAP", "1")
	out3 := applyInjectShaper(context.Background(),
		OpenAIRequest{Model: "claude-haiku-4-5"}, nil)
	if out3.Model != "claude-haiku-4-5" {
		t.Errorf("model hidup ga boleh diremap: got %q", out3.Model)
	}
	_ = store.Settings{}
}
