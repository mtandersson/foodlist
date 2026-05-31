package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/webp"

	"github.com/google/uuid"
	"github.com/jdeng/goheif"
)

// Recipe is the stored representation of a recipe. It is NOT projected from
// events; recipes live in their own JSON files outside the event log.
type Recipe struct {
	ID            string       `json:"id"`
	Title         string       `json:"title"`
	Ingredients   []Ingredient `json:"ingredients"`
	Instructions  []string     `json:"instructions"`
	ImageFilename string       `json:"imageFilename"`
	ImageMIME     string       `json:"imageMime"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

// Ingredient is one row of a recipe ingredient list.
type Ingredient struct {
	Amount *float64 `json:"amount,omitempty"`
	Unit   string   `json:"unit,omitempty"`
	Name   string   `json:"name"`
}

// RecipeMeta is the lightweight projection used by list responses.
type RecipeMeta struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	ImageFilename string    `json:"imageFilename"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Bounds enforced for any persisted recipe. Limits guard against
// LLM-output and PATCH-driven memory/disk DoS.
const (
	maxRecipeTitleLen      = 200
	maxRecipeIngredients   = 50
	maxRecipeInstructions  = 50
	maxRecipeStringLen     = 2000
	maxRecipeMetadataBytes = 256 * 1024
)

// allowedImageMimes maps each accepted sniffed MIME to its canonical
// extension. The map is the only source of truth for the sidecar
// extension; client filenames and Content-Type headers are ignored.
//
// HEIC/HEIF are NOT in this map: they are accepted on upload but
// transcoded to JPEG before storage (see transcodeForStorage). Only
// browser-renderable mimes ever land on disk so we can serve every
// sidecar with a strict Content-Type and no client-side polyfills.
var allowedImageMimes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// uploadOnlyImageMimes lists mimes the API accepts on POST but never
// stores as-is. They go through transcodeForStorage which produces a
// member of allowedImageMimes.
var uploadOnlyImageMimes = map[string]struct{}{
	"image/heic": {},
	"image/heif": {},
}

// ErrRecipeNotFound is returned by RecipeStore when an id does not exist.
var ErrRecipeNotFound = errors.New("recipe not found")

// ErrRecipeInvalid signals a recipe failed structural validation.
var ErrRecipeInvalid = errors.New("recipe invalid")

// ErrUnsupportedImage signals an image MIME type outside the allowlist.
var ErrUnsupportedImage = errors.New("unsupported image type")

// RecipeStore owns the on-disk recipe directory. All paths it touches are
// validated to live under the configured base directory; ids are required
// to be UUIDs so a malicious request cannot escape the directory even if
// the prefix check were bypassed.
type RecipeStore struct {
	mu        sync.Mutex
	baseDir   string
	maxPixels int
	onChanged func(id string, deleted bool)
}

// NewRecipeStore validates and (if necessary) creates the base directory.
// The directory must be inside dataDir to satisfy the path-traversal rule.
func NewRecipeStore(baseDir, dataDir string, maxPixels int) (*RecipeStore, error) {
	if baseDir == "" {
		baseDir = filepath.Join(dataDir, "recipes")
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve recipe dir: %w", err)
	}
	absData, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}
	baseWithSep := absBase + string(os.PathSeparator)
	dataWithSep := absData + string(os.PathSeparator)
	// Path-traversal guard: RECIPE_DIR must live inside DATA_DIR. Same
	// pattern as validateCachePath; failure messages do not include the
	// resolved path to avoid leaking deployment layout.
	if !strings.HasPrefix(baseWithSep, dataWithSep) && absBase != absData {
		return nil, errors.New("recipe dir escapes data dir")
	}
	if err := os.MkdirAll(absBase, 0o755); err != nil {
		return nil, fmt.Errorf("create recipe dir: %w", err)
	}
	if maxPixels <= 0 {
		maxPixels = 24_000_000
	}
	return &RecipeStore{baseDir: absBase, maxPixels: maxPixels}, nil
}

// SetChangeHook registers a callback fired after a successful Save, Update,
// or Delete. The hook is invoked SYNCHRONOUSLY while the store mutex is
// still held - this is what guarantees RecipeChanged broadcasts arrive in
// the same order as the mutations themselves. Holding the mutex during a
// non-blocking enqueue is safe; the hook MUST NOT call back into
// RecipeStore (it would deadlock) and MUST NOT block.
func (s *RecipeStore) SetChangeHook(fn func(id string, deleted bool)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onChanged = fn
}

// MaxImagePixels returns the configured pixel cap.
func (s *RecipeStore) MaxImagePixels() int {
	return s.maxPixels
}

// resolveJSON returns the absolute path to a recipe's JSON file after
// validating the id is a UUID and the resulting path stays inside baseDir.
func (s *RecipeStore) resolveJSON(id string) (string, error) {
	if _, err := uuid.Parse(id); err != nil {
		return "", ErrRecipeNotFound
	}
	p := filepath.Join(s.baseDir, id+".json")
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", ErrRecipeNotFound
	}
	if !strings.HasPrefix(abs, s.baseDir+string(os.PathSeparator)) {
		return "", ErrRecipeNotFound
	}
	return abs, nil
}

// resolveImage returns the absolute path for the recipe's image sidecar
// using the canonical extension derived from MIME.
func (s *RecipeStore) resolveImage(id, mime string) (string, error) {
	ext, ok := allowedImageMimes[mime]
	if !ok {
		return "", ErrUnsupportedImage
	}
	if _, err := uuid.Parse(id); err != nil {
		return "", ErrRecipeNotFound
	}
	p := filepath.Join(s.baseDir, id+ext)
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", ErrRecipeNotFound
	}
	if !strings.HasPrefix(abs, s.baseDir+string(os.PathSeparator)) {
		return "", ErrRecipeNotFound
	}
	return abs, nil
}

// findExistingImage probes the allowed extensions and returns the path
// and MIME of the sidecar image, if any. Used by Get/Delete to avoid
// trusting the recipe's stored ImageFilename for filesystem ops.
func (s *RecipeStore) findExistingImage(id string) (string, string, bool) {
	if _, err := uuid.Parse(id); err != nil {
		return "", "", false
	}
	for mime, ext := range allowedImageMimes {
		p := filepath.Join(s.baseDir, id+ext)
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(abs, s.baseDir+string(os.PathSeparator)) {
			continue
		}
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			return abs, mime, true
		}
	}
	return "", "", false
}

// ValidateAndNormalize enforces field bounds and trims whitespace.
// Returned recipe is safe to persist.
func ValidateAndNormalize(r Recipe) (Recipe, error) {
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		return r, fmt.Errorf("%w: title required", ErrRecipeInvalid)
	}
	if len(r.Title) > maxRecipeTitleLen {
		return r, fmt.Errorf("%w: title too long", ErrRecipeInvalid)
	}
	if len(r.Ingredients) > maxRecipeIngredients {
		return r, fmt.Errorf("%w: too many ingredients", ErrRecipeInvalid)
	}
	if len(r.Instructions) > maxRecipeInstructions {
		return r, fmt.Errorf("%w: too many instructions", ErrRecipeInvalid)
	}
	cleanedIng := make([]Ingredient, 0, len(r.Ingredients))
	for _, ing := range r.Ingredients {
		ing.Name = strings.TrimSpace(ing.Name)
		ing.Unit = strings.TrimSpace(ing.Unit)
		if ing.Name == "" {
			continue
		}
		if len(ing.Name) > maxRecipeStringLen {
			return r, fmt.Errorf("%w: ingredient name too long", ErrRecipeInvalid)
		}
		if len(ing.Unit) > maxRecipeStringLen {
			return r, fmt.Errorf("%w: ingredient unit too long", ErrRecipeInvalid)
		}
		cleanedIng = append(cleanedIng, ing)
	}
	r.Ingredients = cleanedIng
	cleanedSteps := make([]string, 0, len(r.Instructions))
	for _, step := range r.Instructions {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}
		if len(step) > maxRecipeStringLen {
			return r, fmt.Errorf("%w: instruction too long", ErrRecipeInvalid)
		}
		cleanedSteps = append(cleanedSteps, step)
	}
	r.Instructions = cleanedSteps
	return r, nil
}

// SniffImageMIME inspects the leading bytes of imgBytes and returns the
// allowlisted MIME, or ErrUnsupportedImage. Client-supplied Content-Type
// is intentionally not used.
//
// HEIC/HEIF are detected manually because http.DetectContentType has
// no entry for them - it returns application/octet-stream for iPhone
// photos, which would otherwise force a "unsupported image" rejection
// before we even get a chance to transcode.
func SniffImageMIME(imgBytes []byte) (string, error) {
	if len(imgBytes) == 0 {
		return "", ErrUnsupportedImage
	}
	if mime := sniffISOBMFFBrand(imgBytes); mime != "" {
		return mime, nil
	}
	// http.DetectContentType uses up to 512 bytes per the docs.
	mime := http.DetectContentType(imgBytes)
	// DetectContentType may tack on charset; strip it.
	if idx := strings.IndexByte(mime, ';'); idx >= 0 {
		mime = strings.TrimSpace(mime[:idx])
	}
	if _, ok := allowedImageMimes[mime]; ok {
		return mime, nil
	}
	if _, ok := uploadOnlyImageMimes[mime]; ok {
		return mime, nil
	}
	return "", ErrUnsupportedImage
}

// sniffISOBMFFBrand recognises HEIC/HEIF by the ISO Base Media File
// Format `ftyp` box brand. Returns "" if the bytes are not an ISOBMFF
// container or the brand is unknown to us.
//
// Brand reference (ISO/IEC 23008-12):
//   - heic, heix, heim, heis, hevc, hevx — HEIC variants
//   - mif1, msf1                         — generic HEIF image/sequence
//
// We deliberately do NOT match avif here; goheif cannot decode AVIF
// and we'd rather reject up front than mis-route to a transcoder that
// will fail.
func sniffISOBMFFBrand(b []byte) string {
	if len(b) < 12 {
		return ""
	}
	if !bytes.Equal(b[4:8], []byte("ftyp")) {
		return ""
	}
	switch string(b[8:12]) {
	case "heic", "heix", "heim", "heis", "hevc", "hevx", "mif1", "msf1":
		return "image/heic"
	}
	return ""
}

// transcodeForStorage converts an upload to one of the storable mimes
// in allowedImageMimes, returning the (possibly identical) byte slice
// and its final mime. JPEG/PNG/WebP pass through unchanged so we never
// re-encode and lose quality. HEIC/HEIF are decoded with goheif and
// re-encoded as quality-90 JPEG.
//
// HEIC EXIF orientation is honored: iPhone photos in portrait mode
// store landscape pixels with an Orientation tag (0x0112) telling
// the renderer to rotate. goheif.Decode does not auto-rotate, and
// re-encoding strips EXIF entirely (which is what we want - HEIC
// EXIF can include GPS coordinates and camera serials we have no
// business persisting on a recipe sidecar). To keep the photo
// upright we apply the rotation/flip to the pixel buffer BEFORE
// encoding the JPEG.
//
// Returns the original slice (same backing array) on the passthrough
// path so callers can rely on identity to detect "no work was done".
func transcodeForStorage(imgBytes []byte, mime string) ([]byte, string, error) {
	if _, ok := allowedImageMimes[mime]; ok {
		return imgBytes, mime, nil
	}
	if _, ok := uploadOnlyImageMimes[mime]; !ok {
		return nil, "", ErrUnsupportedImage
	}

	// Pull EXIF first; ExtractExif advances the reader, so we use a
	// fresh bytes.Reader for Decode.
	exifBytes, _ := goheif.ExtractExif(bytes.NewReader(imgBytes))
	img, err := goheif.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, "", fmt.Errorf("%w: cannot decode heic", ErrUnsupportedImage)
	}
	if orientation := parseExifOrientation(exifBytes); orientation > 1 {
		img = rotateForOrientation(img, orientation)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, "", fmt.Errorf("%w: cannot re-encode heic as jpeg", ErrUnsupportedImage)
	}
	return buf.Bytes(), "image/jpeg", nil
}

// parseExifOrientation returns the EXIF Orientation tag (0x0112)
// from a TIFF-formatted EXIF block, or 0 if not present / invalid.
//
// We deliberately do NOT pull in a full EXIF parser library: we
// only need one tag, the upstream blob comes from a single source
// (goheif.ExtractExif on iPhone photos), and a 30-line manual
// parser cannot have its own CVE / supply-chain footprint.
//
// The exif argument may optionally start with the "Exif\x00\x00"
// prefix that JPEG APP1 markers use; we strip it so the same parser
// works for raw HEIC EXIF and for JPEG-extracted EXIF in the future.
func parseExifOrientation(exif []byte) int {
	if len(exif) >= 6 && bytes.Equal(exif[:6], []byte{'E', 'x', 'i', 'f', 0, 0}) {
		exif = exif[6:]
	}
	if len(exif) < 8 {
		return 0
	}
	var bo binary.ByteOrder
	switch string(exif[0:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	if bo.Uint16(exif[2:4]) != 0x002A {
		return 0
	}
	ifdOffset := bo.Uint32(exif[4:8])
	if int(ifdOffset)+2 > len(exif) {
		return 0
	}
	count := int(bo.Uint16(exif[ifdOffset : ifdOffset+2]))
	start := int(ifdOffset) + 2
	for i := 0; i < count; i++ {
		base := start + i*12
		if base+12 > len(exif) {
			return 0
		}
		tag := bo.Uint16(exif[base : base+2])
		if tag != 0x0112 {
			continue
		}
		typ := bo.Uint16(exif[base+2 : base+4])
		if typ != 3 {
			// Type SHORT is the only legal type for this tag.
			return 0
		}
		// Value is in the first 2 bytes of the 4-byte value/offset
		// field because count=1 and SHORT=2 bytes fit inline.
		return int(bo.Uint16(exif[base+8 : base+10]))
	}
	return 0
}

// rotateForOrientation returns a new image with the EXIF orientation
// applied. Orientation values are defined by the EXIF spec:
//
//	1 = identity
//	2 = mirror horizontal
//	3 = rotate 180
//	4 = mirror vertical
//	5 = transpose (mirror horizontal + rotate 90 CW)
//	6 = rotate 90 CW   (most common - iPhone portrait)
//	7 = transverse (mirror horizontal + rotate 90 CCW)
//	8 = rotate 90 CCW
//
// 0 (absent) and 1 (identity) return the source unchanged. Anything
// outside 1..8 is treated as identity rather than panicking - we
// would rather show a sideways recipe than crash the upload path.
func rotateForOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	// 5..8 swap dimensions; 2..4 keep them.
	var dst *image.RGBA
	if orientation >= 5 {
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
	} else {
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
	}

	// Draw the source into an RGBA buffer first so .At() lookups
	// stay cheap when the source is a YCbCr (typical from JPEG).
	srcRGBA := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(srcRGBA, srcRGBA.Bounds(), src, b.Min, draw.Src)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := srcRGBA.RGBAAt(x, y)
			var nx, ny int
			switch orientation {
			case 2: // mirror horizontal
				nx, ny = w-1-x, y
			case 3: // rotate 180
				nx, ny = w-1-x, h-1-y
			case 4: // mirror vertical
				nx, ny = x, h-1-y
			case 5: // transpose
				nx, ny = y, x
			case 6: // rotate 90 CW
				nx, ny = h-1-y, x
			case 7: // transverse
				nx, ny = h-1-y, w-1-x
			case 8: // rotate 90 CCW
				nx, ny = y, w-1-x
			}
			dst.SetRGBA(nx, ny, c)
		}
	}
	return dst
}

// CheckImageBounds validates the decoded dimensions of the image without
// fully decoding the pixel buffer (uses image.DecodeConfig). Pixel count
// must stay under maxPixels to bound memory use during LLM calls and
// thumbnails. Returns the resolved MIME on success.
func (s *RecipeStore) CheckImageBounds(imgBytes []byte) (string, error) {
	mime, err := SniffImageMIME(imgBytes)
	if err != nil {
		return "", err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(imgBytes))
	if err != nil {
		return "", fmt.Errorf("%w: cannot decode image config", ErrUnsupportedImage)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return "", fmt.Errorf("%w: invalid image dimensions", ErrUnsupportedImage)
	}
	pixels := int64(cfg.Width) * int64(cfg.Height)
	if pixels > int64(s.maxPixels) {
		return "", fmt.Errorf("%w: image too large", ErrUnsupportedImage)
	}
	return mime, nil
}

// List returns all recipe metadata, newest first.
func (s *RecipeStore) List() ([]RecipeMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read recipe dir: %w", err)
	}
	metas := make([]RecipeMeta, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		if _, err := uuid.Parse(id); err != nil {
			continue
		}
		r, err := s.readUnlocked(id)
		if err != nil {
			continue
		}
		metas = append(metas, RecipeMeta{
			ID:            r.ID,
			Title:         r.Title,
			ImageFilename: r.ImageFilename,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		})
	}
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].CreatedAt.After(metas[j].CreatedAt)
	})
	return metas, nil
}

func (s *RecipeStore) readUnlocked(id string) (Recipe, error) {
	jsonPath, err := s.resolveJSON(id)
	if err != nil {
		return Recipe{}, err
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Recipe{}, ErrRecipeNotFound
		}
		return Recipe{}, fmt.Errorf("read recipe: %w", err)
	}
	var r Recipe
	if err := json.Unmarshal(data, &r); err != nil {
		return Recipe{}, fmt.Errorf("decode recipe: %w", err)
	}
	return r, nil
}

// Get returns a single recipe.
func (s *RecipeStore) Get(id string) (Recipe, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readUnlocked(id)
}

// Save writes a new recipe with id, atomically writing both the JSON
// metadata and the image sidecar. On any error mid-write, both temp files
// are cleaned up so the directory never contains a half-saved recipe.
func (s *RecipeStore) Save(r Recipe, imageBytes []byte, mime string) (Recipe, error) {
	if _, ok := allowedImageMimes[mime]; !ok {
		return Recipe{}, ErrUnsupportedImage
	}
	if _, err := uuid.Parse(r.ID); err != nil {
		return Recipe{}, fmt.Errorf("%w: invalid id", ErrRecipeInvalid)
	}
	cleaned, err := ValidateAndNormalize(r)
	if err != nil {
		return Recipe{}, err
	}
	now := time.Now().UTC()
	if cleaned.CreatedAt.IsZero() {
		cleaned.CreatedAt = now
	}
	cleaned.UpdatedAt = now
	cleaned.ImageMIME = mime
	cleaned.ImageFilename = cleaned.ID + allowedImageMimes[mime]

	jsonPath, err := s.resolveJSON(cleaned.ID)
	if err != nil {
		return Recipe{}, err
	}
	imagePath, err := s.resolveImage(cleaned.ID, mime)
	if err != nil {
		return Recipe{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := writeAtomic(imagePath, imageBytes); err != nil {
		return Recipe{}, fmt.Errorf("write image: %w", err)
	}
	jsonBytes, err := json.MarshalIndent(cleaned, "", "  ")
	if err != nil {
		_ = os.Remove(imagePath)
		return Recipe{}, fmt.Errorf("marshal recipe: %w", err)
	}
	if err := writeAtomic(jsonPath, jsonBytes); err != nil {
		_ = os.Remove(imagePath)
		return Recipe{}, fmt.Errorf("write recipe: %w", err)
	}

	if s.onChanged != nil {
		s.onChanged(cleaned.ID, false)
	}
	return cleaned, nil
}

// Update overwrites the metadata-only fields of an existing recipe. The
// sidecar image is left untouched; image replacement is not supported in
// v1 to avoid orphaned files.
func (s *RecipeStore) Update(id string, mutate func(Recipe) (Recipe, error)) (Recipe, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Recipe{}, ErrRecipeNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.readUnlocked(id)
	if err != nil {
		return Recipe{}, err
	}
	updated, err := mutate(current)
	if err != nil {
		return Recipe{}, err
	}
	updated.ID = current.ID
	updated.CreatedAt = current.CreatedAt
	updated.ImageFilename = current.ImageFilename
	updated.ImageMIME = current.ImageMIME
	cleaned, err := ValidateAndNormalize(updated)
	if err != nil {
		return Recipe{}, err
	}
	cleaned.UpdatedAt = time.Now().UTC()

	jsonPath, err := s.resolveJSON(id)
	if err != nil {
		return Recipe{}, err
	}
	jsonBytes, err := json.MarshalIndent(cleaned, "", "  ")
	if err != nil {
		return Recipe{}, fmt.Errorf("marshal recipe: %w", err)
	}
	if err := writeAtomic(jsonPath, jsonBytes); err != nil {
		return Recipe{}, fmt.Errorf("write recipe: %w", err)
	}
	if s.onChanged != nil {
		s.onChanged(id, false)
	}
	return cleaned, nil
}

// Delete removes both the JSON and image sidecar. Missing files are
// tolerated so a partially-saved recipe can still be cleaned up.
func (s *RecipeStore) Delete(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrRecipeNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	jsonPath, err := s.resolveJSON(id)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(jsonPath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return ErrRecipeNotFound
		}
	}
	if err := os.Remove(jsonPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete recipe: %w", err)
	}
	if imagePath, _, ok := s.findExistingImage(id); ok {
		_ = os.Remove(imagePath)
	}
	if s.onChanged != nil {
		s.onChanged(id, true)
	}
	return nil
}

// ReadImage returns the raw image bytes and stored MIME for a recipe.
func (s *RecipeStore) ReadImage(id string) ([]byte, string, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, "", ErrRecipeNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	abs, mime, ok := s.findExistingImage(id)
	if !ok {
		return nil, "", ErrRecipeNotFound
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", ErrRecipeNotFound
		}
		return nil, "", fmt.Errorf("read image: %w", err)
	}
	return data, mime, nil
}

// writeAtomic writes data to path via a tmp file + rename.
// On any error, the tmp file is removed.
func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
