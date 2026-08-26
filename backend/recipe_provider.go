package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	recipeProviderOpenAICompatible = "openai_compatible"
	recipeProviderOpenAIAPI        = "openai_api"
	recipeProviderCodexOAuth       = "experimental_codex_oauth"
	defaultOpenAIRecipeModel       = "gpt-5.6-sol"
)

type recipeParserSelection struct {
	Parser   RecipeImageParser
	Provider string
	Model    string
}

func recipeLLMRequested(cfg Config) bool {
	return strings.TrimSpace(cfg.RecipeLLMProvider) != "" ||
		strings.TrimSpace(cfg.RecipeLLMBaseURL) != "" ||
		strings.TrimSpace(cfg.RecipeLLMAPIKey) != "" ||
		strings.TrimSpace(cfg.RecipeLLMModel) != "" ||
		strings.TrimSpace(cfg.RecipeLLMAuthFile) != ""
}

func recipeLLMSecurityConfigured(cfg Config) bool {
	return strings.TrimSpace(cfg.SharedSecret) != "" && len(cfg.CIDRWhitelist) > 0
}

func selectRecipeImageParser(cfg Config) (recipeParserSelection, error) {
	provider := strings.TrimSpace(cfg.RecipeLLMProvider)
	baseURL := strings.TrimSpace(cfg.RecipeLLMBaseURL)
	apiKey := strings.TrimSpace(cfg.RecipeLLMAPIKey)
	model := strings.TrimSpace(cfg.RecipeLLMModel)

	if provider == "" {
		configured := 0
		for _, value := range []string{baseURL, apiKey, model} {
			if value != "" {
				configured++
			}
		}
		if configured == 0 {
			return recipeParserSelection{}, nil
		}
		if configured != 3 {
			return recipeParserSelection{}, fmt.Errorf("%w: incomplete legacy recipe llm configuration", ErrLLMConfigInvalid)
		}
		parser, err := NewRecipeLLMClient(baseURL, apiKey, model)
		return recipeParserSelection{Parser: parser, Provider: recipeProviderOpenAICompatible, Model: model}, err
	}

	switch provider {
	case recipeProviderOpenAICompatible:
		if baseURL == "" || apiKey == "" || model == "" {
			return recipeParserSelection{}, fmt.Errorf("%w: openai_compatible requires base url, api key, and model", ErrLLMConfigInvalid)
		}
		parser, err := NewRecipeLLMClient(baseURL, apiKey, model)
		return recipeParserSelection{Parser: parser, Provider: provider, Model: model}, err

	case recipeProviderOpenAIAPI:
		if apiKey == "" {
			return recipeParserSelection{}, fmt.Errorf("%w: openai_api requires api key", ErrLLMConfigInvalid)
		}
		if model == "" {
			model = defaultOpenAIRecipeModel
		}
		parser, err := NewOpenAIResponsesRecipeClient(apiKey, model)
		return recipeParserSelection{Parser: parser, Provider: provider, Model: model}, err

	case recipeProviderCodexOAuth:
		if model == "" {
			model = defaultOpenAIRecipeModel
		}
		authFile := strings.TrimSpace(cfg.RecipeLLMAuthFile)
		if authFile == "" {
			authFile = filepath.Join(cfg.DataDir, "secrets", "recipe-openai-auth.json")
		}
		manager, err := NewCodexOAuthManager(authFile, cfg.DataDir)
		if err != nil {
			return recipeParserSelection{}, err
		}
		parser, err := NewCodexOAuthRecipeClient(manager, model, version)
		if err != nil {
			_ = manager.Close()
			return recipeParserSelection{}, err
		}
		return recipeParserSelection{Parser: parser, Provider: provider, Model: model}, nil

	default:
		return recipeParserSelection{}, fmt.Errorf("%w: unknown recipe llm provider", ErrLLMConfigInvalid)
	}
}
