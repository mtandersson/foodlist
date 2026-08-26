package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectRecipeImageParserPrecedence(t *testing.T) {
	t.Run("unset and empty disables parsing", func(t *testing.T) {
		selection, err := selectRecipeImageParser(Config{DataDir: t.TempDir()})
		require.NoError(t, err)
		require.Nil(t, selection.Parser)
	})

	t.Run("unset complete legacy triplet", func(t *testing.T) {
		selection, err := selectRecipeImageParser(Config{
			DataDir: t.TempDir(), RecipeLLMBaseURL: "https://example.com/v1",
			RecipeLLMAPIKey: "secret", RecipeLLMModel: "vision-model",
		})
		require.NoError(t, err)
		require.IsType(t, &RecipeLLMClient{}, selection.Parser)
		require.Equal(t, recipeProviderOpenAICompatible, selection.Provider)
		require.NotNil(t, selection.Parser.(*RecipeLLMClient).http.CheckRedirect)
	})

	t.Run("unset partial legacy disables", func(t *testing.T) {
		selection, err := selectRecipeImageParser(Config{DataDir: t.TempDir(), RecipeLLMAPIKey: "secret"})
		require.Error(t, err)
		require.Nil(t, selection.Parser)
	})

	t.Run("explicit official ignores irrelevant base url and defaults model", func(t *testing.T) {
		selection, err := selectRecipeImageParser(Config{
			DataDir: t.TempDir(), RecipeLLMProvider: recipeProviderOpenAIAPI,
			RecipeLLMBaseURL: "https://ignored.invalid", RecipeLLMAPIKey: "secret",
		})
		require.NoError(t, err)
		require.IsType(t, &responsesRecipeClient{}, selection.Parser)
		require.Equal(t, defaultOpenAIRecipeModel, selection.Model)
	})

	t.Run("explicit provider wins over complete legacy triplet", func(t *testing.T) {
		selection, err := selectRecipeImageParser(Config{
			DataDir: t.TempDir(), RecipeLLMProvider: recipeProviderOpenAIAPI,
			RecipeLLMBaseURL: "https://legacy.example/v1", RecipeLLMAPIKey: "secret",
			RecipeLLMModel: "explicit-model",
		})
		require.NoError(t, err)
		require.Equal(t, recipeProviderOpenAIAPI, selection.Provider)
		require.Equal(t, "explicit-model", selection.Model)
	})

	t.Run("unknown provider", func(t *testing.T) {
		selection, err := selectRecipeImageParser(Config{DataDir: t.TempDir(), RecipeLLMProvider: "mystery"})
		require.Error(t, err)
		require.Nil(t, selection.Parser)
	})
}

func TestRecipeLLMSecurityConfigured(t *testing.T) {
	require.False(t, recipeLLMSecurityConfigured(Config{}))
	require.False(t, recipeLLMSecurityConfigured(Config{SharedSecret: "secret"}))
	require.False(t, recipeLLMSecurityConfigured(Config{CIDRWhitelist: []string{"127.0.0.1/32"}}))
	require.True(t, recipeLLMSecurityConfigured(Config{SharedSecret: "secret", CIDRWhitelist: []string{"127.0.0.1/32"}}))
}
