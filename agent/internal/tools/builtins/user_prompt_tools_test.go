package builtins

import (
	"testing"

	"flowork-gui/internal/tools"
)

func TestUserPromptToolsRegistered(t *testing.T) {
	// Call Init() to register all builtin tools statically defined in builtins.go
	// (Normally called once during main.go startup)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Init() already called elsewhere: %v", r)
		}
	}()
	Init()

	requiredTools := []string{
		"instinct_recall",
		"mistake_recall",
		"interaction_recall",
		"memory_get",
		"skill_search",
		"graph_recall",
		"brain_search",
		"cognitive_tensions",
		"cognitive_resolve",
		"web_search",
		"webfetch",
		"task_list",
		"task_run",
		"tool_search",
	}

	for _, name := range requiredTools {
		tool, ok := tools.Lookup(name)
		if !ok {
			t.Errorf("Tool %q NOT registered in the agent registry!", name)
		} else {
			t.Logf("Tool %q is registered. Schema Description: %q", name, tool.Schema().Description)
		}
	}
}
