package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/a3tai/library/internal/aimeta"
	"github.com/a3tai/library/internal/embedder"
	"github.com/a3tai/library/internal/library"
)

type Settings struct {
	LMStudioURL           string `json:"lmstudioURL"`
	LMStudioEmbedModel    string `json:"lmstudioEmbedModel"`
	LMStudioChatModel     string `json:"lmstudioChatModel"`
	LMStudioKeyConfigured bool   `json:"lmstudioKeyConfigured"`
	MCPPort               int    `json:"mcpPort"`
	DBPath                string `json:"dbPath"`
}

// SettingsInput is what the UI sends back on Save. APIKey uses a sentinel
// so the user can clear the key explicitly (empty string) without that
// being confused with "the form didn't carry the key" — see UpdateSettings.
type SettingsInput struct {
	LMStudioURL        string `json:"lmstudioURL"`
	LMStudioEmbedModel string `json:"lmstudioEmbedModel"`
	LMStudioChatModel  string `json:"lmstudioChatModel"`
	// APIKey is *string so the form can distinguish "leave existing value
	// alone" (nil) from "explicitly clear it" (pointer to "").
	APIKey *string `json:"apiKey,omitempty"`
}

// Settings returns the current effective configuration for the settings
// panel. Stored values win; env vars + built-in defaults fill any blanks.
func (s *LibraryService) Settings() (Settings, error) {
	ctx := context.Background()
	stored, err := s.store.GetSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	emb := embedder.NewFromConfig(embedder.Resolve(stored))
	ai := aimeta.NewFromConfig(aimeta.Resolve(stored))
	dbPath, _ := library.DefaultDBPath()
	return Settings{
		LMStudioURL:           emb.BaseURL,
		LMStudioEmbedModel:    emb.Model,
		LMStudioChatModel:     ai.Model,
		LMStudioKeyConfigured: strings.TrimSpace(emb.APIKey) != "",
		MCPPort:               s.MCPStatus().Port,
		DBPath:                dbPath,
	}, nil
}

// UpdateSettings writes the supplied values into the settings table and
// rebuilds the LM Studio clients atomically. Empty string values fall back
// to env + defaults (i.e. they clear any user-set override). Returns the
// new effective Settings.
func (s *LibraryService) UpdateSettings(input SettingsInput) (Settings, error) {
	ctx := context.Background()
	kv := map[string]string{
		embedder.SettingURL:        strings.TrimSpace(input.LMStudioURL),
		embedder.SettingEmbedModel: strings.TrimSpace(input.LMStudioEmbedModel),
		aimeta.SettingChatModel:    strings.TrimSpace(input.LMStudioChatModel),
	}
	if input.APIKey != nil {
		kv[embedder.SettingAPIKey] = strings.TrimSpace(*input.APIKey)
	}
	if err := s.store.SetSettings(ctx, kv); err != nil {
		return Settings{}, err
	}
	stored, err := s.store.GetSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	// Swap in fresh clients. The embedder is behind an atomic Provider
	// (MCP server + search path), and aimeta is replaced under aiMu since
	// the chat path reads through s.aimeta().
	s.embed.Set(embedder.NewFromConfig(embedder.Resolve(stored)))
	s.aiMu.Lock()
	s.ai = aimeta.NewFromConfig(aimeta.Resolve(stored))
	s.aiMu.Unlock()
	// Drop the per-model context-budget cache so the next chat turn
	// re-discovers the new model's window via /api/v0/models.
	s.chatBudgetMu.Lock()
	s.chatBudgetCache = nil
	s.chatBudgetMu.Unlock()
	// Nudge the embedding scanner so a model change kicks off backfill
	// for any books that don't yet have vectors under the new model.
	s.kickEmbedder()
	return s.Settings()
}

// TestLMStudio probes the configured endpoint with /v1/models. Returns the
// model id list on success so the settings panel can confirm what LM Studio
// is actually serving.
func (s *LibraryService) TestLMStudio() (bool, error) {
	client := s.embed.Get()
	if client == nil {
		return false, errors.New("embedder not configured")
	}
	ctx := context.Background()
	return client.Available(ctx), nil
}

// AppVersion is the version string surfaced in the sidebar footer / about
// menu. Update on each release. Returned as a method so the frontend can
// read it through the existing service binding without a separate constant
// export.
const AppVersion = "1.0.0"

// Version returns the AppVersion constant. Bound to the frontend so the
// sidebar footer can render it without a hard-coded JS copy.
func (s *LibraryService) Version() string {
	return AppVersion
}

// ListLMStudioModels returns the model ids advertised by /v1/models so the
// settings panel can populate model dropdowns. Returns an empty slice (not
// an error) when the endpoint isn't reachable so the UI can degrade to a
// free-form text field cleanly.
func (s *LibraryService) ListLMStudioModels() ([]string, error) {
	client := s.embed.Get()
	if client == nil {
		return []string{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	models, err := client.ListModels(ctx)
	if err != nil {
		// Treat fetch errors as "no models known" — the UI falls back to a
		// text input. Surface the message for diagnostic display, not to
		// block save.
		return []string{}, nil
	}
	return models, nil
}
