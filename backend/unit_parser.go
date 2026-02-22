package main

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

//go:embed data/swedish_units.json
var swedishUnitsJSON []byte

// UnitLexicon holds the Swedish unit and modifier lexicon.
type UnitLexicon struct {
	Modifiers struct {
		Approximate   []string       `json:"approximate"`
		QuantityWords map[string]any `json:"quantity_words"`
	} `json:"modifiers"`
	Volume    []string `json:"volume"`
	Weight    []string `json:"weight"`
	Packaging []string `json:"packaging"`
}

// allUnits returns a flattened, deduplicated list of all units, sorted by length descending.
func (l *UnitLexicon) allUnits() []string {
	seen := make(map[string]bool)
	var units []string
	for _, u := range append(append(l.Volume, l.Weight...), l.Packaging...) {
		ul := strings.ToLower(strings.TrimSpace(u))
		if ul != "" && !seen[ul] {
			seen[ul] = true
			units = append(units, ul)
		}
	}
	sort.Slice(units, func(i, j int) bool { return len(units[i]) > len(units[j]) })
	return units
}

// allModifiers returns approximate + quantity_words keys, sorted by length descending.
func (l *UnitLexicon) allModifiers() []string {
	seen := make(map[string]bool)
	for _, m := range l.Modifiers.Approximate {
		ml := strings.ToLower(strings.TrimSpace(m))
		if ml != "" && !seen[ml] {
			seen[ml] = true
		}
	}
	for m := range l.Modifiers.QuantityWords {
		ml := strings.ToLower(strings.TrimSpace(m))
		if ml != "" && !seen[ml] {
			seen[ml] = true
		}
	}
	var mods []string
	for m := range seen {
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool { return len(mods[i]) > len(mods[j]) })
	return mods
}

// quantityWordValue returns the numeric value for halv/en/ett, or nil for lite.
func (l *UnitLexicon) quantityWordValue(mod string) *float64 {
	if l.Modifiers.QuantityWords == nil {
		return nil
	}
	v, ok := l.Modifiers.QuantityWords[strings.ToLower(mod)]
	if !ok {
		return nil
	}
	if v == nil {
		return nil // lite
	}
	switch n := v.(type) {
	case float64:
		return &n
	case int:
		f := float64(n)
		return &f
	}
	return nil
}

var (
	lexicon     *UnitLexicon
	lexiconErr  error
	numberRegex = regexp.MustCompile(`^\d+(?:[.,]\d+)?`)
)

func init() {
	lexicon = &UnitLexicon{}
	lexiconErr = json.Unmarshal(swedishUnitsJSON, lexicon)
}

// TokenType represents the type of a token.
type TokenType string

const (
	TokenMODIFIER TokenType = "MODIFIER"
	TokenNUMBER   TokenType = "NUMBER"
	TokenUNIT     TokenType = "UNIT"
	TokenLPAREN   TokenType = "LPAREN"
	TokenRPAREN   TokenType = "RPAREN"
	TokenWORD     TokenType = "WORD"
)

// Token represents a single token from the tokenizer.
type Token struct {
	Type  TokenType
	Value string
}

// fuzzyMatchUnit finds the best lexicon match for input using Levenshtein distance.
// Returns (canonical unit, true) if a unique best match within threshold exists.
// Rejects matches where input is much longer than unit (e.g. "mjölk" must not match "ml").
func fuzzyMatchUnit(input string, units []string) (string, bool) {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return "", false
	}
	// Exact match first
	for _, u := range units {
		if input == u {
			return u, true
		}
	}
	// Fuzzy: Levenshtein with length-scaled threshold
	// Require input length close to unit length to avoid "mjölk"->"ml", "ölk"->"l"
	threshold := 2
	if len(input) > 5 {
		threshold = 3
	}
	var best string
	bestDist := threshold + 1
	tie := false
	for _, u := range units {
		// Reject if input much longer than unit (e.g. "mjölk"->"ml", "ölk"->"l")
		// Allow +2 for typos like "gramm"->"gram", "burka"->"burk"
		if len(input) > len(u)+2 {
			continue
		}
		d := levenshteinDistance(input, u)
		if d < bestDist {
			bestDist = d
			best = u
			tie = false
		} else if d == bestDist && best != "" && u != best {
			tie = true
		}
	}
	if best != "" && !tie && bestDist <= threshold {
		return best, true
	}
	return "", false
}

// preNormalize applies "halv" -> 0.5, "en"/"ett" -> 1 before tokenizing.
func preNormalize(input string, mods []string) string {
	s := " " + strings.TrimSpace(input) + " "
	// halv
	s = regexp.MustCompile(`(?i)\bhalv\s+`).ReplaceAllString(s, " 0.5 ")
	// en, ett (only when followed by space and likely a unit - we do simple replacement)
	s = regexp.MustCompile(`(?i)\ben\s+`).ReplaceAllString(s, " 1 ")
	s = regexp.MustCompile(`(?i)\bett\s+`).ReplaceAllString(s, " 1 ")
	return strings.TrimSpace(s)
}

// Tokenize splits input into tokens.
func Tokenize(input string, lex *UnitLexicon) []Token {
	if lex == nil || lexiconErr != nil {
		lex = lexicon
	}
	if lex == nil {
		return nil
	}
	input = preNormalize(input, lex.allModifiers())
	units := lex.allUnits()
	mods := lex.allModifiers()

	var tokens []Token
	i := 0
	for i < len(input) {
		// Skip spaces
		for i < len(input) && (input[i] == ' ' || input[i] == '\t') {
			i++
		}
		if i >= len(input) {
			break
		}
		rest := input[i:]

		// MODIFIER (only at start of remaining, after potential space skip)
		matched := false
		for _, m := range mods {
			if len(rest) >= len(m) && strings.EqualFold(rest[:len(m)], m) {
				// Word boundary: end of string or followed by space/punctuation
				after := len(rest) > len(m)
				if after {
					next := rest[len(m)]
					if next != ' ' && next != '\t' && next != '(' {
						continue
					}
				}
				tokens = append(tokens, Token{TokenMODIFIER, m})
				i += len(m)
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// NUMBER
		if loc := numberRegex.FindStringIndex(rest); loc != nil {
			numStr := rest[loc[0]:loc[1]]
			numStr = strings.Replace(numStr, ",", ".", 1)
			tokens = append(tokens, Token{TokenNUMBER, numStr})
			i += loc[1]
			continue
		}

		// LPAREN, RPAREN
		if rest[0] == '(' {
			tokens = append(tokens, Token{TokenLPAREN, "("})
			i++
			continue
		}
		if rest[0] == ')' {
			tokens = append(tokens, Token{TokenRPAREN, ")"})
			i++
			continue
		}

		// UNIT (exact or fuzzy) - try longest first
		unitMatched := false
		for _, u := range units {
			if len(rest) >= len(u) && strings.EqualFold(rest[:len(u)], u) {
				after := len(rest) > len(u)
				if after {
					next := rest[len(u)]
					if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || (next >= '0' && next <= '9') || next == 'å' || next == 'ä' || next == 'ö' {
						continue // word continues, not a unit boundary
					}
				}
				tokens = append(tokens, Token{TokenUNIT, u})
				i += len(u)
				unitMatched = true
				break
			}
		}
		if !unitMatched {
			// Fuzzy unit match - try to consume a word and check (use runes for UTF-8)
			wordEnd := 0
			for wordEnd < len(rest) {
				r, size := utf8.DecodeRuneInString(rest[wordEnd:])
				if size == 0 {
					break
				}
				if unicode.IsLetter(r) {
					wordEnd += size
				} else {
					break
				}
			}
			if wordEnd > 0 {
				word := rest[:wordEnd]
				if canon, ok := fuzzyMatchUnit(word, units); ok {
					tokens = append(tokens, Token{TokenUNIT, canon})
					i += wordEnd
					unitMatched = true
				}
			}
		}
		if unitMatched {
			continue
		}

		// WORD (use runes for UTF-8, e.g. mjölk)
		wordEnd := 0
		for wordEnd < len(rest) {
			r, size := utf8.DecodeRuneInString(rest[wordEnd:])
			if size == 0 {
				break
			}
			if unicode.IsLetter(r) || (r >= '0' && r <= '9') {
				wordEnd += size
			} else {
				break
			}
		}
		if wordEnd > 0 {
			tokens = append(tokens, Token{TokenWORD, rest[:wordEnd]})
			i += wordEnd
		} else {
			i++ // skip unknown char
		}
	}
	return tokens
}

// ParseResult holds the result of parsing an ingredient input.
type ParseResult struct {
	Name          string
	Count         *float64
	Unit          *string
	OriginalInput string
}

// ParseIngredientInput parses natural language ingredient input (e.g. "2l mjölk", "1 burk tomater (300g)").
func ParseIngredientInput(input string) ParseResult {
	original := strings.TrimSpace(input)
	if original == "" {
		return ParseResult{OriginalInput: original}
	}

	if lexicon == nil || lexiconErr != nil {
		return ParseResult{Name: original, OriginalInput: original}
	}

	tokens := Tokenize(original, lexicon)
	if len(tokens) == 0 {
		return ParseResult{Name: original, OriginalInput: original}
	}

	// Try patterns in order
	if r := matchModifierName(tokens, lexicon); r != nil {
		r.OriginalInput = original
		return *r
	}
	if r := matchPkgSizeName(tokens); r != nil {
		r.OriginalInput = original
		return *r
	}
	if r := matchParentheticalName(tokens); r != nil {
		r.OriginalInput = original
		return *r
	}
	if r := matchParentheticalSize(tokens); r != nil {
		r.OriginalInput = original
		return *r
	}
	if r := matchPrefix(tokens, lexicon); r != nil {
		r.OriginalInput = original
		return *r
	}
	if r := matchSuffix(tokens); r != nil {
		r.OriginalInput = original
		return *r
	}
	if r := matchUnitOnly(tokens, lexicon); r != nil {
		r.OriginalInput = original
		return *r
	}

	return ParseResult{Name: original, OriginalInput: original}
}

// matchModifierName: MODIFIER WORD+ -> name only (lite mjöl)
func matchModifierName(tokens []Token, lex *UnitLexicon) *ParseResult {
	if len(tokens) < 2 {
		return nil
	}
	if tokens[0].Type != TokenMODIFIER {
		return nil
	}
	// lite has no numeric value
	if lex.quantityWordValue(tokens[0].Value) != nil {
		return nil
	}
	var words []string
	for i := 1; i < len(tokens) && tokens[i].Type == TokenWORD; i++ {
		words = append(words, tokens[i].Value)
	}
	if len(words) == 0 {
		return nil
	}
	name := strings.Join(words, " ")
	return &ParseResult{Name: name}
}

// matchPkgSizeName: NUMBER UNIT NUMBER UNIT WORD+
func matchPkgSizeName(tokens []Token) *ParseResult {
	if len(tokens) < 5 {
		return nil
	}
	i := 0
	if tokens[i].Type != TokenNUMBER || tokens[i+1].Type != TokenUNIT || tokens[i+2].Type != TokenNUMBER || tokens[i+3].Type != TokenUNIT {
		return nil
	}
	c1, _ := strconv.ParseFloat(tokens[i].Value, 64)
	u1 := tokens[i+1].Value
	i += 4
	var words []string
	for i < len(tokens) && tokens[i].Type == TokenWORD {
		words = append(words, tokens[i].Value)
		i++
	}
	if len(words) == 0 {
		return nil
	}
	cu1 := c1
	un := u1
	return &ParseResult{
		Name:  strings.Join(words, " "),
		Count: &cu1,
		Unit:  &un,
	}
}

// matchParentheticalName: NUMBER UNIT WORD+ LPAREN NUMBER UNIT RPAREN
func matchParentheticalName(tokens []Token) *ParseResult {
	if len(tokens) < 7 {
		return nil
	}
	i := 0
	if tokens[i].Type != TokenNUMBER || tokens[i+1].Type != TokenUNIT {
		return nil
	}
	c, _ := strconv.ParseFloat(tokens[i].Value, 64)
	u := tokens[i+1].Value
	i += 2
	var words []string
	for i < len(tokens) && tokens[i].Type == TokenWORD {
		words = append(words, tokens[i].Value)
		i++
	}
	if len(words) == 0 || i+4 > len(tokens) {
		return nil
	}
	if tokens[i].Type != TokenLPAREN || tokens[i+1].Type != TokenNUMBER || tokens[i+2].Type != TokenUNIT || tokens[i+3].Type != TokenRPAREN {
		return nil
	}
	return &ParseResult{
		Name:  strings.Join(words, " "),
		Count: &c,
		Unit:  &u,
	}
}

// matchParentheticalSize: NUMBER UNIT LPAREN NUMBER UNIT RPAREN WORD+
func matchParentheticalSize(tokens []Token) *ParseResult {
	if len(tokens) < 7 {
		return nil
	}
	i := 0
	if tokens[i].Type != TokenNUMBER || tokens[i+1].Type != TokenUNIT || tokens[i+2].Type != TokenLPAREN || tokens[i+3].Type != TokenNUMBER || tokens[i+4].Type != TokenUNIT || tokens[i+5].Type != TokenRPAREN {
		return nil
	}
	c, _ := strconv.ParseFloat(tokens[i].Value, 64)
	u := tokens[i+1].Value
	i += 6
	var words []string
	for i < len(tokens) && tokens[i].Type == TokenWORD {
		words = append(words, tokens[i].Value)
		i++
	}
	if len(words) == 0 {
		return nil
	}
	return &ParseResult{
		Name:  strings.Join(words, " "),
		Count: &c,
		Unit:  &u,
	}
}

// matchPrefix: MODIFIER? NUMBER (WORD* UNIT WORD+)
// Allows filler words between number and unit (e.g. "2 stora burkar tomater")
func matchPrefix(tokens []Token, lex *UnitLexicon) *ParseResult {
	i := 0
	if i < len(tokens) && tokens[i].Type == TokenMODIFIER {
		i++ // skip modifier (ca, etc.)
	}
	if i >= len(tokens) || tokens[i].Type != TokenNUMBER {
		return nil
	}
	c, _ := strconv.ParseFloat(tokens[i].Value, 64)
	i++
	// Skip optional WORDs until we find UNIT
	for i < len(tokens) && tokens[i].Type == TokenWORD {
		i++
	}
	if i >= len(tokens) || tokens[i].Type != TokenUNIT {
		return nil
	}
	u := tokens[i].Value
	i++
	var words []string
	for i < len(tokens) && tokens[i].Type == TokenWORD {
		words = append(words, tokens[i].Value)
		i++
	}
	if len(words) == 0 {
		return nil
	}
	return &ParseResult{
		Name:  strings.Join(words, " "),
		Count: &c,
		Unit:  &u,
	}
}

// matchSuffix: WORD+ NUMBER UNIT
func matchSuffix(tokens []Token) *ParseResult {
	if len(tokens) < 3 {
		return nil
	}
	// Find NUMBER UNIT at end
	var words []string
	i := 0
	for i < len(tokens) && tokens[i].Type == TokenWORD {
		words = append(words, tokens[i].Value)
		i++
	}
	if len(words) == 0 || i+2 > len(tokens) {
		return nil
	}
	if tokens[i].Type != TokenNUMBER || tokens[i+1].Type != TokenUNIT {
		return nil
	}
	if i+2 != len(tokens) {
		return nil
	}
	c, _ := strconv.ParseFloat(tokens[i].Value, 64)
	u := tokens[i+1].Value
	return &ParseResult{
		Name:  strings.Join(words, " "),
		Count: &c,
		Unit:  &u,
	}
}

// matchUnitOnly: MODIFIER? NUMBER UNIT (no name)
func matchUnitOnly(tokens []Token, lex *UnitLexicon) *ParseResult {
	i := 0
	if i < len(tokens) && tokens[i].Type == TokenMODIFIER {
		i++ // skip modifier
	}
	if i+2 != len(tokens) {
		return nil
	}
	if tokens[i].Type != TokenNUMBER || tokens[i+1].Type != TokenUNIT {
		return nil
	}
	c, _ := strconv.ParseFloat(tokens[i].Value, 64)
	u := tokens[i+1].Value
	return &ParseResult{Count: &c, Unit: &u}
}
