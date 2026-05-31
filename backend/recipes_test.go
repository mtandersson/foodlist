package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTestPNG returns a small valid PNG image of the given dimensions.
func makeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestRecipeStore_Save_Get_Delete(t *testing.T) {
	tmp := t.TempDir()
	store, err := NewRecipeStore(filepath.Join(tmp, "recipes"), tmp, 1_000_000)
	require.NoError(t, err)

	imgBytes := makeTestPNG(t, 32, 32)
	mime, err := SniffImageMIME(imgBytes)
	require.NoError(t, err)
	require.Equal(t, "image/png", mime)

	r := Recipe{
		ID:    uuid.NewString(),
		Title: "Pannkakor",
		Sections: []RecipeSection{{
			Ingredients: []Ingredient{
				{Name: "Mjölk", Unit: "dl", Amount: ptrFloat(3)},
				{Name: "Mjöl", Unit: "dl", Amount: ptrFloat(2)},
			},
			Instructions: []string{"Vispa", "Stek"},
		}},
	}

	saved, err := store.Save(r, imgBytes, mime)
	require.NoError(t, err)
	require.Equal(t, r.ID, saved.ID)
	require.NotZero(t, saved.CreatedAt)
	require.Equal(t, "image/png", saved.ImageMIME)
	require.Equal(t, saved.ID+".png", saved.ImageFilename)

	got, err := store.Get(saved.ID)
	require.NoError(t, err)
	require.Equal(t, "Pannkakor", got.Title)

	imgOut, mimeOut, err := store.ReadImage(saved.ID)
	require.NoError(t, err)
	require.Equal(t, mime, mimeOut)
	require.Equal(t, imgBytes, imgOut)

	require.NoError(t, store.Delete(saved.ID))
	_, err = store.Get(saved.ID)
	require.ErrorIs(t, err, ErrRecipeNotFound)
}

func TestRecipeStore_PathTraversal(t *testing.T) {
	tmp := t.TempDir()
	store, err := NewRecipeStore(filepath.Join(tmp, "recipes"), tmp, 1_000_000)
	require.NoError(t, err)

	// All of these must be rejected with ErrRecipeNotFound.
	cases := []string{
		"../../../etc/passwd",
		"..",
		"foo/bar",
		"00000000-0000-0000-0000-00000000000Z",
		"",
	}
	for _, id := range cases {
		_, err := store.Get(id)
		require.ErrorIs(t, err, ErrRecipeNotFound, "id %q should be rejected", id)
		_, _, err = store.ReadImage(id)
		require.ErrorIs(t, err, ErrRecipeNotFound, "image id %q should be rejected", id)
	}
}

func TestRecipeStore_DirEscapesDataDir(t *testing.T) {
	tmp := t.TempDir()
	other := t.TempDir()
	_, err := NewRecipeStore(other, tmp, 1_000_000)
	require.Error(t, err)
}

func TestSniffImageMIME(t *testing.T) {
	png := makeTestPNG(t, 16, 16)
	mime, err := SniffImageMIME(png)
	require.NoError(t, err)
	require.Equal(t, "image/png", mime)

	_, err = SniffImageMIME([]byte("hello world"))
	require.ErrorIs(t, err, ErrUnsupportedImage)

	_, err = SniffImageMIME(nil)
	require.ErrorIs(t, err, ErrUnsupportedImage)
}

// TestSniffImageMIME_HEIC ensures HEIC/HEIF magic bytes resolve to a
// recognisable MIME so the transcode pipeline can pick them up.
// http.DetectContentType has no HEIC entry, so the sniffer must
// special-case the ISO Base Media `ftyp` brand.
func TestSniffImageMIME_HEIC(t *testing.T) {
	heicBytes, err := os.ReadFile(filepath.Join("testdata", "sample.heic"))
	require.NoError(t, err)

	mime, err := SniffImageMIME(heicBytes)
	require.NoError(t, err, "real HEIC fixture must be recognised")
	require.Equal(t, "image/heic", mime)

	// Synthetic minimal HEIC header (ftyp + brand) - enough for the
	// sniffer even without a real decoded payload.
	for _, brand := range []string{"heic", "heix", "mif1", "msf1"} {
		hdr := append([]byte{0, 0, 0, 0x20}, []byte("ftyp"+brand)...)
		hdr = append(hdr, make([]byte, 32)...)
		got, err := SniffImageMIME(hdr)
		require.NoError(t, err, "brand %q must sniff", brand)
		require.Equal(t, "image/heic", got)
	}
}

// TestTranscodeForStorage_HEICToJPEG decodes a real HEIC file and
// re-encodes it as JPEG so the rest of the pipeline (CheckImageBounds,
// LLM, sidecar storage) keeps working with one of the canonical
// storable mimes. The returned bytes must themselves sniff back to
// image/jpeg so the transcode is genuine, not a passthrough.
func TestTranscodeForStorage_HEICToJPEG(t *testing.T) {
	heicBytes, err := os.ReadFile(filepath.Join("testdata", "sample.heic"))
	require.NoError(t, err)

	out, mime, err := transcodeForStorage(heicBytes, "image/heic")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", mime)

	// Must be a real JPEG (not the original HEIC bytes), and must be
	// decodable so we know the JPEG is well-formed.
	sniffed, err := SniffImageMIME(out)
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", sniffed)
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	require.Greater(t, cfg.Width, 0)
	require.Greater(t, cfg.Height, 0)
}

// TestTranscodeForStorage_PassthroughForJPEG checks the helper is a
// no-op for already-storable mimes; it must NOT re-encode a JPEG and
// degrade quality, and it must NOT change the byte slice identity.
func TestTranscodeForStorage_PassthroughForJPEG(t *testing.T) {
	// PNG that we'd just pass through.
	pngBytes := makeTestPNG(t, 32, 32)
	out, mime, err := transcodeForStorage(pngBytes, "image/png")
	require.NoError(t, err)
	require.Equal(t, "image/png", mime)
	require.Equal(t, &pngBytes[0], &out[0], "passthrough should not copy")
}

func TestRecipeStore_PixelCap(t *testing.T) {
	tmp := t.TempDir()
	// Cap deliberately tiny so even a 32x32 image fails.
	store, err := NewRecipeStore(filepath.Join(tmp, "recipes"), tmp, 16)
	require.NoError(t, err)

	imgBytes := makeTestPNG(t, 32, 32)
	_, err = store.CheckImageBounds(imgBytes)
	require.ErrorIs(t, err, ErrUnsupportedImage)
}

func TestValidateAndNormalize(t *testing.T) {
	r := Recipe{
		Title: " Pannkakor  ",
		Sections: []RecipeSection{{
			Name: "  ",
			Ingredients: []Ingredient{
				{Name: " "}, // empty after trim, dropped
				{Name: "Mjölk", Unit: "dl"},
			},
			Instructions: []string{"  Vispa  ", "", "Stek"},
		}},
	}
	cleaned, err := ValidateAndNormalize(r)
	require.NoError(t, err)
	require.Equal(t, "Pannkakor", cleaned.Title)
	require.Len(t, cleaned.Sections, 1)
	require.Len(t, cleaned.Sections[0].Ingredients, 1)
	require.Equal(t, []string{"Vispa", "Stek"}, cleaned.Sections[0].Instructions)
	require.Equal(t, "", cleaned.Sections[0].Name)

	// Title required.
	_, err = ValidateAndNormalize(Recipe{Title: "  "})
	require.ErrorIs(t, err, ErrRecipeInvalid)

	// Too many ingredients across sections (summed).
	tooMany := Recipe{Title: "ok", Sections: []RecipeSection{{
		Ingredients: make([]Ingredient, maxRecipeIngredients+1),
	}}}
	// Fill names so the empty-name filter doesn't drop them silently.
	for i := range tooMany.Sections[0].Ingredients {
		tooMany.Sections[0].Ingredients[i] = Ingredient{Name: fmt.Sprintf("ing-%d", i)}
	}
	_, err = ValidateAndNormalize(tooMany)
	require.ErrorIs(t, err, ErrRecipeInvalid)
}

func ptrFloat(f float64) *float64 { return &f }

// --- Cook session tests ---------------------------------------------------

func TestCookSessions_CheckUncheckReset(t *testing.T) {
	c := NewCookSessions()

	steps := c.Check("r1", 0)
	assert.Equal(t, []int{0}, steps)
	steps = c.Check("r1", 2)
	assert.Equal(t, []int{0, 2}, steps)

	steps = c.Uncheck("r1", 0)
	assert.Equal(t, []int{2}, steps)

	c.Reset("r1")
	assert.Empty(t, c.Snapshot())
}

func TestCookSessions_PruneAbove(t *testing.T) {
	c := NewCookSessions()
	c.Check("r1", 0)
	c.Check("r1", 3)
	c.Check("r1", 5)

	steps, changed := c.PruneAbove("r1", 4)
	assert.True(t, changed)
	assert.Equal(t, []int{0, 3}, steps)

	// Already pruned: same call should report no change.
	_, changed = c.PruneAbove("r1", 4)
	assert.False(t, changed)
}

func TestCookSessions_Concurrent(t *testing.T) {
	c := NewCookSessions()
	var wg sync.WaitGroup
	const n = 100
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.Check("r", i)
		}(i)
	}
	wg.Wait()
	snap := c.Snapshot()
	assert.Len(t, snap["r"], n)
}

// --- HTTP layer tests -----------------------------------------------------

func newTestRecipeAPI(t *testing.T) (*RecipeAPI, *http.ServeMux, *RecipeStore) {
	t.Helper()
	tmp := t.TempDir()
	store, err := NewRecipeStore(filepath.Join(tmp, "recipes"), tmp, 1_000_000)
	require.NoError(t, err)
	api := NewRecipeAPI(store, nil, nil, "", 100)
	mux := http.NewServeMux()
	api.Register(mux, func(h http.Handler) http.Handler { return h })
	return api, mux, store
}

func multipartCreate(t *testing.T, imgBytes []byte, meta any) (*http.Request, error) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("image", "test.png")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, bytes.NewReader(imgBytes)); err != nil {
		return nil, err
	}
	if meta != nil {
		metaBytes, _ := json.Marshal(meta)
		if err := w.WriteField("metadata", string(metaBytes)); err != nil {
			return nil, err
		}
	}
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recipes", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req, nil
}

func TestRecipeAPI_CreateGetDelete(t *testing.T) {
	_, mux, _ := newTestRecipeAPI(t)
	imgBytes := makeTestPNG(t, 64, 48)

	req, err := multipartCreate(t, imgBytes, map[string]any{
		"title": "Min rätt",
		"sections": []any{
			map[string]any{
				"ingredients":  []any{map[string]any{"name": "Salt", "unit": "tsk", "amount": 1}},
				"instructions": []string{"Krydda"},
			},
		},
	})
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var detail recipeDetailResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &detail))
	require.Equal(t, "Min rätt", detail.Recipe.Title)

	// GET it back.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/recipes/"+detail.Recipe.ID, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	require.Equal(t, http.StatusOK, getRR.Code)

	// Image with nosniff + correct content-type.
	imgReq := httptest.NewRequest(http.MethodGet, "/api/v1/recipes/"+detail.Recipe.ID+"/image", nil)
	imgRR := httptest.NewRecorder()
	mux.ServeHTTP(imgRR, imgReq)
	require.Equal(t, http.StatusOK, imgRR.Code)
	require.Equal(t, "image/png", imgRR.Header().Get("Content-Type"))
	require.Equal(t, "nosniff", imgRR.Header().Get("X-Content-Type-Options"))

	// Delete.
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/recipes/"+detail.Recipe.ID, nil)
	delRR := httptest.NewRecorder()
	mux.ServeHTTP(delRR, delReq)
	require.Equal(t, http.StatusNoContent, delRR.Code)
}

func TestRecipeAPI_NonUUIDIs404(t *testing.T) {
	_, mux, _ := newTestRecipeAPI(t)
	// A plain non-UUID id reaches our handler and must 404. Path-traversal
	// payloads like "../../etc/passwd" are first cleaned by net/http's
	// path canonicalization (returning a 307 redirect to the cleaned
	// path), so they never reach our handler at all - which is fine
	// from a security standpoint and tested separately at the store
	// layer in TestRecipeStore_PathTraversal.
	for _, id := range []string{"abc", "not-a-uuid", "00000000-0000-0000-0000-00000000000Z"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/recipes/"+id, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		require.Equal(t, http.StatusNotFound, rr.Code, "id %q must 404", id)
	}
}

func TestRecipeAPI_RejectsUnsupportedImage(t *testing.T) {
	_, mux, _ := newTestRecipeAPI(t)
	req, err := multipartCreate(t, []byte("not an image"), map[string]any{
		"title": "x",
	})
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRecipeAPI_OversizedMetadataIsRejected(t *testing.T) {
	_, mux, _ := newTestRecipeAPI(t)
	imgBytes := makeTestPNG(t, 16, 16)
	huge := strings.Repeat("a", recipeMetadataMaxBytes+1)
	req, err := multipartCreate(t, imgBytes, map[string]any{"title": huge})
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	require.GreaterOrEqual(t, rr.Code, 400)
}

// TestParseExifOrientation_RecognisesIPhonePortrait builds a
// minimal EXIF blob (the same shape as what an iPhone embeds) with
// orientation tag 0x0112 = 6 ("rotate 90 CW") and asserts the
// parser pulls it out. iPhone photos in portrait mode store
// landscape pixels and rely on this tag to display upright; if we
// don't honor it, the recipe sidecar shows up sideways.
func TestParseExifOrientation_RecognisesIPhonePortrait(t *testing.T) {
	for _, tc := range []struct {
		name        string
		bo          string
		orientation uint16
	}{
		{"little_endian_orientation_6", "II", 6},
		{"big_endian_orientation_8", "MM", 8},
		{"with_exif_header_prefix_3", "II", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exif := buildSyntheticExif(t, tc.bo, tc.orientation)
			if tc.name == "with_exif_header_prefix_3" {
				exif = append([]byte("Exif\x00\x00"), exif...)
			}
			got := parseExifOrientation(exif)
			require.Equal(t, int(tc.orientation), got)
		})
	}
}

func TestParseExifOrientation_AbsentOrInvalidIsZero(t *testing.T) {
	require.Equal(t, 0, parseExifOrientation(nil))
	require.Equal(t, 0, parseExifOrientation([]byte("")))
	require.Equal(t, 0, parseExifOrientation([]byte("not exif")))
	// Truncated mid-IFD must not panic and must report 0.
	exif := buildSyntheticExif(t, "II", 6)
	require.Equal(t, 0, parseExifOrientation(exif[:8]))
}

// TestRotateForOrientation_Six rotates a 2x3 marker image and pins
// the corner pixels so we know the transform is correct, not just
// "different". Orientation 6 means "rotate 90 CW", so the original
// (0,0) pixel ends up at (h-1, 0) of the output (which has swapped
// dimensions). Without this, an iPhone landscape-stored portrait
// would still serve sideways.
func TestRotateForOrientation_Six(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 3))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})   // top-left  red
	src.Set(1, 0, color.RGBA{0, 255, 0, 255})   // top-right green
	src.Set(0, 2, color.RGBA{0, 0, 255, 255})   // bot-left  blue
	src.Set(1, 2, color.RGBA{255, 255, 0, 255}) // bot-right yellow

	out := rotateForOrientation(src, 6)
	b := out.Bounds()
	require.Equal(t, 3, b.Dx(), "rotated width should equal source height")
	require.Equal(t, 2, b.Dy(), "rotated height should equal source width")
	// 90 CW: top-left of src -> top-right of dst
	r, g, _, _ := out.At(2, 0).RGBA()
	require.Equal(t, uint32(0xffff), r, "top-right of rotated image should be the original top-left red")
	require.Equal(t, uint32(0), g)
	// Bot-right of src (yellow) -> bot-left of dst
	r2, g2, _, _ := out.At(0, 1).RGBA()
	require.Equal(t, uint32(0xffff), r2)
	require.Equal(t, uint32(0xffff), g2)
}

func TestRotateForOrientation_PassthroughForOneOrZero(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 5))
	src.Set(0, 0, color.RGBA{1, 2, 3, 255})
	for _, o := range []int{0, 1} {
		out := rotateForOrientation(src, o)
		require.Equal(t, src.Bounds(), out.Bounds())
		require.Equal(t, src.At(0, 0), out.At(0, 0))
	}
}

// buildSyntheticExif constructs the smallest valid EXIF/TIFF block
// containing a single Orientation entry. Keeping this in test code
// (rather than vendoring a fixture file) lets us check both byte
// orders and avoids committing yet another binary to the tree.
func buildSyntheticExif(t *testing.T, bo string, orientation uint16) []byte {
	t.Helper()
	var byteOrder binary.ByteOrder
	switch bo {
	case "II":
		byteOrder = binary.LittleEndian
	case "MM":
		byteOrder = binary.BigEndian
	default:
		t.Fatalf("bad byte order %q", bo)
	}
	buf := make([]byte, 0, 32)
	buf = append(buf, byte(bo[0]), byte(bo[1]))
	tmp := make([]byte, 2)
	byteOrder.PutUint16(tmp, 0x002A)
	buf = append(buf, tmp...)
	tmp4 := make([]byte, 4)
	byteOrder.PutUint32(tmp4, 8) // IFD0 starts at offset 8
	buf = append(buf, tmp4...)
	// IFD0: 1 entry
	byteOrder.PutUint16(tmp, 1)
	buf = append(buf, tmp...)
	// Entry: tag=0x0112 type=3 count=1 value=orientation
	byteOrder.PutUint16(tmp, 0x0112)
	buf = append(buf, tmp...)
	byteOrder.PutUint16(tmp, 3)
	buf = append(buf, tmp...)
	byteOrder.PutUint32(tmp4, 1)
	buf = append(buf, tmp4...)
	byteOrder.PutUint16(tmp, orientation)
	buf = append(buf, tmp...)
	buf = append(buf, 0, 0) // pad value field to 4 bytes
	// Next-IFD offset = 0
	buf = append(buf, 0, 0, 0, 0)
	return buf
}

// TestRecipeAPI_AcceptsHEIC pushes a real iPhone-style HEIC file
// through the full create endpoint and asserts the server transcodes
// it to JPEG before storing - the user never sees an "unsupported
// image" error and the sidecar served back is a real JPEG so any
// browser can render it.
func TestRecipeAPI_AcceptsHEIC(t *testing.T) {
	// Use a generous pixel cap; the HEIC fixture is camera-sized.
	tmp := t.TempDir()
	store, err := NewRecipeStore(filepath.Join(tmp, "recipes"), tmp, 50_000_000)
	require.NoError(t, err)
	api := NewRecipeAPI(store, nil, nil, "", 100)
	mux := http.NewServeMux()
	api.Register(mux, func(h http.Handler) http.Handler { return h })

	heicBytes, err := os.ReadFile(filepath.Join("testdata", "sample.heic"))
	require.NoError(t, err)

	req, err := multipartCreate(t, heicBytes, map[string]any{
		"title": "Foto från iPhone",
		"sections": []any{
			map[string]any{
				"name":         "",
				"ingredients":  []any{map[string]any{"name": "Salt"}},
				"instructions": []string{"Smaka av"},
			},
		},
	})
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var detail recipeDetailResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &detail))
	require.Equal(t, "image/jpeg", detail.Recipe.ImageMIME,
		"HEIC uploads must be transcoded to a browser-renderable mime")
	require.True(t, strings.HasSuffix(detail.Recipe.ImageFilename, ".jpg"))

	// Round-trip the image through the API and check it really is JPEG.
	data, mime, err := store.ReadImage(detail.Recipe.ID)
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", mime)
	sniffed, err := SniffImageMIME(data)
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", sniffed)
}

// TestRecipeAPI_ListEmpty_ProductionMux reproduces the routing layout
// used by main.go: a static-file server mounted at the path prefix plus
// the recipe API mounted underneath it. A regression where the recipe
// route is shadowed by the file server (or where Content-Type drifts to
// text/html) causes the frontend to surface
//
//	"JSON.parse: unexpected character at line 1 column 1 of the JSON data"
//
// even on the empty-list happy path. The test asserts JSON content-type
// AND a parseable body so a future routing reorder cannot regress it
// silently.
func TestRecipeAPI_ListEmpty_ProductionMux(t *testing.T) {
	cases := []struct {
		name       string
		pathPrefix string // trailing slash like main.go's pathPrefix
	}{
		{"no_secret", "/"},
		{"with_secret", "/dev-secret/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			store, err := NewRecipeStore(filepath.Join(tmp, "recipes"), tmp, 1_000_000)
			require.NoError(t, err)

			mux := http.NewServeMux()

			// Mirror main.go: serve a static dir under the path prefix.
			staticDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(staticDir, "index.html"),
				[]byte("<!doctype html><title>shell</title>"), 0o644))
			fileServer := http.FileServer(http.Dir(staticDir))
			if tc.pathPrefix != "/" {
				prefixToStrip := tc.pathPrefix[:len(tc.pathPrefix)-1]
				fileServer = http.StripPrefix(prefixToStrip, fileServer)
			}
			mux.Handle(tc.pathPrefix, fileServer)

			// Recipe API mounted with the trimmed prefix, exactly like main.go.
			api := NewRecipeAPI(store, nil, nil, strings.TrimSuffix(tc.pathPrefix, "/"), 100)
			api.Register(mux, func(h http.Handler) http.Handler { return h })

			listURL := strings.TrimSuffix(tc.pathPrefix, "/") + "/api/v1/recipes"
			req := httptest.NewRequest(http.MethodGet, listURL, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code,
				"GET %s on empty store should be 200, got %d body=%q",
				listURL, rr.Code, rr.Body.String())
			require.Contains(t, rr.Header().Get("Content-Type"), "application/json",
				"GET %s should serve JSON, got Content-Type=%q body=%q",
				listURL, rr.Header().Get("Content-Type"), rr.Body.String())

			var resp recipeListResponse
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp),
				"empty-list body must be parseable JSON, got %q", rr.Body.String())
			require.NotNil(t, resp.Recipes,
				"recipes field must be a JSON array (got null) on empty store")
			require.Empty(t, resp.Recipes)
		})
	}
}

// Verify writeAtomic does not leave .tmp files on success and cleans up
// on failure (we simulate failure by writing into a non-existent dir).
func TestWriteAtomic_CleansUp(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "out.txt")
	require.NoError(t, writeAtomic(dest, []byte("hello")))
	_, err := os.Stat(dest + ".tmp")
	require.True(t, os.IsNotExist(err), "tmp file must not remain on success")

	// Failure case: non-existent parent dir leaves no tmp file behind.
	missing := filepath.Join(tmp, "no-such-dir", "x.txt")
	require.Error(t, writeAtomic(missing, []byte("nope")))
	_, err = os.Stat(missing + ".tmp")
	require.True(t, os.IsNotExist(err), "tmp file must not remain on failure")
}
