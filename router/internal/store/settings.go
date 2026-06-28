// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type Settings struct {
	RequireLogin bool           `json:"requireLogin"`
	AuthMode     string         `json:"authMode"`
	Password     string         `json:"password,omitempty"`
	OidcConfig   map[string]any `json:"oidcConfig,omitempty"`

	RequireApiKey bool `json:"requireApiKey"`

	DefaultModel     string `json:"defaultModel"`
	FallbackStrategy string `json:"fallbackStrategy"`

	RtkTokenSaver bool `json:"rtkTokenSaver"`

	IntentRouting IntentRouting `json:"intentRouting"`

	CostRouting CostRouting `json:"costRouting"`

	CavemanLevel string `json:"cavemanLevel,omitempty"`

	ClaudeCliBypass ClaudeCliBypass `json:"claudeCliBypass"`

	Budget Budget `json:"budget"`

	Brain BrainConfig `json:"brain"`

	TunnelUrl    string `json:"tunnelUrl,omitempty"`
	TailscaleUrl string `json:"tailscaleUrl,omitempty"`
}

type BrainConfig struct {
	Enabled           bool     `json:"enabled"`
	Model             string   `json:"model"`
	DBPath            string   `json:"dbPath,omitempty"`
	Mode              string   `json:"mode"`
	Wings             []string `json:"wings,omitempty"`
	TopK              int      `json:"topK"`
	MaxSnippetChars   int      `json:"maxSnippetChars"`
	Skills            bool     `json:"skills"`
	SkillTopK         int      `json:"skillTopK"`
	MaxSkillBodyChars int      `json:"maxSkillBodyChars"`
	Record            bool     `json:"record"`

	AlwaysOn bool `json:"alwaysOn"`

	InjectConstitution   bool `json:"injectConstitution"`
	ConstitutionTopK     int  `json:"constitutionTopK"`
	ConstitutionMaxChars int  `json:"constitutionMaxChars"`
}

type ClaudeCliBypass struct {
	Enabled        bool     `json:"enabled"`
	SkipPatterns   []string `json:"skipPatterns,omitempty"`
	CcFilterNaming bool     `json:"ccFilterNaming,omitempty"`
}

type IntentRouting struct {
	Enabled         bool     `json:"enabled"`
	PrivatePatterns []string `json:"privatePatterns"`
	PrivateTag      string   `json:"privateTag"`
}

type CostRouting struct {
	Enabled            bool `json:"enabled"`
	CheapMaxChars      int  `json:"cheapMaxChars"`
	StandardMaxChars   int  `json:"standardMaxChars"`
	StrongOnCode       bool `json:"strongOnCode"`
	StrongOnToolUse    bool `json:"strongOnToolUse"`
	StrongMinMessages  int  `json:"strongMinMessages"`
	HonorExplicitModel bool `json:"honorExplicitModel"`
}

type Budget struct {
	Enforce       bool    `json:"enforce"`
	DailyCapUsd   float64 `json:"dailyCapUsd"`
	MonthlyCapUsd float64 `json:"monthlyCapUsd"`
	WarnUsd       float64 `json:"warnUsd"`
}

func defaultSettings() Settings {
	return Settings{
		RequireLogin:     false,
		AuthMode:         "none",
		DefaultModel:     "claude-haiku-4-5",
		FallbackStrategy: "priority_ordered",

		Budget: Budget{
			Enforce:       false,
			DailyCapUsd:   2.0,
			MonthlyCapUsd: 60.0,
			WarnUsd:       0.5,
		},

		Brain: BrainConfig{
			Enabled:              true,
			Model:                "flowork-brain",
			Mode:                 "augment",
			TopK:                 5,
			MaxSnippetChars:      600,
			Skills:               true,
			SkillTopK:            3,
			MaxSkillBodyChars:    700, // cap default anti-muntah (skill gede 24×11KB → +286% tok). 0=legacy uncap.
			AlwaysOn:             true,
			InjectConstitution:   true,
			ConstitutionTopK:     20,
			ConstitutionMaxChars: 600,
		},

		ClaudeCliBypass: ClaudeCliBypass{
			Enabled:        true,
			CcFilterNaming: false,
		},

		CostRouting: CostRouting{
			Enabled:            true,
			CheapMaxChars:      2000,
			StandardMaxChars:   10000,
			StrongOnCode:       true,
			StrongOnToolUse:    true,
			StrongMinMessages:  6,
			HonorExplicitModel: true,
		},
	}
}

func LoadSettings(d *sql.DB) (*Settings, error) {
	row := d.QueryRow(`SELECT data FROM settings WHERE id = 1`)
	var raw string
	if err := row.Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			s := defaultSettings()
			return &s, nil
		}
		return nil, fmt.Errorf("settings scan: %w", err)
	}
	var s Settings
	if err := json.Unmarshal([]byte(raw), &s); err != nil {

		s = defaultSettings()
	}

	if s.AuthMode == "" {
		s.AuthMode = "none"
	}
	if s.DefaultModel == "" {
		s.DefaultModel = "claude-haiku-4-5"
	}
	if s.FallbackStrategy == "" {
		s.FallbackStrategy = "priority_ordered"
	}

	if s.CostRouting == (CostRouting{}) {
		s.CostRouting = defaultSettings().CostRouting
	}

	if !s.ClaudeCliBypass.Enabled && len(s.ClaudeCliBypass.SkipPatterns) == 0 && !s.ClaudeCliBypass.CcFilterNaming {
		s.ClaudeCliBypass = defaultSettings().ClaudeCliBypass
	}

	preMigration := s.Brain.ConstitutionTopK == 0 && s.Brain.ConstitutionMaxChars == 0
	if s.Brain.ConstitutionTopK == 0 {
		s.Brain.ConstitutionTopK = defaultSettings().Brain.ConstitutionTopK
	}
	if s.Brain.ConstitutionMaxChars == 0 {
		s.Brain.ConstitutionMaxChars = defaultSettings().Brain.ConstitutionMaxChars
	}
	if s.Brain.SkillTopK == 0 {
		s.Brain.SkillTopK = defaultSettings().Brain.SkillTopK
	}
	allNewBooleansFalse := !s.Brain.AlwaysOn && !s.Brain.InjectConstitution && !s.Brain.Skills
	if preMigration || allNewBooleansFalse {

		d := defaultSettings().Brain
		s.Brain.AlwaysOn = d.AlwaysOn
		s.Brain.InjectConstitution = d.InjectConstitution
		s.Brain.Skills = d.Skills
		if preMigration {
			s.Brain.Enabled = d.Enabled
		}
	}
	return &s, nil
}

func SaveSettings(d *sql.DB, s *Settings) error {

	if s.RequireLogin && s.AuthMode == "password" && s.Password == "" {
		s.RequireLogin = false
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	_, err = d.Exec(`INSERT INTO settings (id, data) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET data = excluded.data`, string(raw))
	if err != nil {
		return fmt.Errorf("upsert settings: %w", err)
	}
	return nil
}

func PatchSettings(d *sql.DB, patch map[string]any) (*Settings, error) {
	current, err := LoadSettings(d)
	if err != nil {
		return nil, err
	}

	curJSON, _ := json.Marshal(current)
	curMap := map[string]any{}
	_ = json.Unmarshal(curJSON, &curMap)
	for k, v := range patch {
		if v == nil {
			continue
		}

		if k == "password" {
			continue
		}
		curMap[k] = v
	}
	mergedJSON, _ := json.Marshal(curMap)
	var merged Settings
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		return nil, fmt.Errorf("patch merge: %w", err)
	}
	if err := SaveSettings(d, &merged); err != nil {
		return nil, err
	}
	return &merged, nil
}
