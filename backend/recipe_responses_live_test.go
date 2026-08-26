//go:build live

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLiveCodexRecipe makes an account-scoped network call and may refresh the
// supplied OAuth token file. The live build tag keeps it entirely outside the
// normal test suite. Run it only with explicit credentials and image paths:
//
//	go test -tags=live -run '^TestLiveCodexRecipe$' -v
func TestLiveCodexRecipe(t *testing.T) {
	authPath := os.Getenv("RECIPE_LLM_AUTH_FILE")
	imagePath := os.Getenv("FOODLIST_LIVE_RECIPE_IMAGE")
	if authPath == "" || imagePath == "" {
		t.Fatal("RECIPE_LLM_AUTH_FILE and FOODLIST_LIVE_RECIPE_IMAGE are required")
	}
	image, err := os.ReadFile(imagePath)
	require.NoError(t, err)
	mime, err := SniffImageMIME(image)
	require.NoError(t, err)
	image, mime, err = transcodeForStorage(image, mime)
	require.NoError(t, err)
	dataDir := filepath.Dir(filepath.Dir(authPath))
	manager, err := NewCodexOAuthManager(authPath, dataDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Close() })
	parser, err := NewCodexOAuthRecipeClient(manager, "gpt-5.6-sol", "live-test")
	require.NoError(t, err)
	recipe, err := parser.ParseImage(context.Background(), image, mime)
	if err != nil {
		var llmErr *RecipeLLMError
		if errors.As(err, &llmErr) && llmErr.cause != nil {
			t.Fatalf("live recipe parse failed (%s): %v", llmErr.Kind, llmErr.cause)
		}
		t.Fatal(err)
	}
	require.NotEmpty(t, recipe.Title)
	require.NotEmpty(t, recipe.Sections)
}
