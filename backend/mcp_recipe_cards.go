package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/jpeg"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/image/draw"
)

type recipeCardsList struct {
	Enabled bool         `json:"enabled"`
	Recipes []RecipeMeta `json:"recipes"`
}

// Decoding one maximum-size stored photo can allocate roughly 100 MB.
// Bound concurrent MCP thumbnail reads across all sessions before loading bytes.
var recipeThumbnailSlot = make(chan struct{}, 1)

//go:embed mcp_app_dist/recipes.html
var recipeCardsAppHTML string

func recipeCardsAppResource() *mcp.ReadResourceResult {
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
		URI: mcpRecipesAppURI, MIMEType: "text/html;profile=mcp-app",
		Text: recipeCardsAppHTML, Meta: shoppingAppMeta(),
	}}}
}

// makeRecipeThumbnail bounds both dimensions to 256 px while preserving
// aspect ratio. The MCP resource carries image bytes only when read.
func makeRecipeThumbnail(data []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, ErrUnsupportedImage
	}
	const maxSide = 256
	if w > maxSide || h > maxSide {
		if w >= h {
			h = max(1, h*maxSide/w)
			w = maxSide
		} else {
			w = max(1, w*maxSide/h)
			h = maxSide
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 78}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
