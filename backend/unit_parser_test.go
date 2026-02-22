package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLexiconLoaded(t *testing.T) {
	require.Nil(t, lexiconErr, "lexicon should load without error")
	require.NotNil(t, lexicon, "lexicon should be non-nil")
	if lexicon != nil {
		units := lexicon.allUnits()
		require.NotEmpty(t, units, "lexicon should have units")
		n := 5
		if len(units) < n {
			n = len(units)
		}
		t.Logf("lexicon has %d units, first few: %v", len(units), units[:n])
	}
}

func TestParseIngredientInput(t *testing.T) {
	if lexicon == nil || lexiconErr != nil {
		t.Fatalf("lexicon not loaded (err=%v) - cannot run parser tests", lexiconErr)
	}
	tests := []struct {
		input    string
		wantName string
		wantCnt  *float64
		wantUnit *string
	}{
		{"2 gram mjöl", "mjöl", floatPtr(2), strPtr("gram")},
		{"1,5 dl mjölk", "mjölk", floatPtr(1.5), strPtr("dl")},
		{"2burka 300gramm krossade tomater", "krossade tomater", floatPtr(2), strPtr("burka")},
		{"lite mjöl", "mjöl", nil, nil},
		{"ca 2 dl mjölk", "mjölk", floatPtr(2), strPtr("dl")},
		{"2l mjölk", "mjölk", floatPtr(2), strPtr("l")},
		{"mjölk 2l", "mjölk", floatPtr(2), strPtr("l")},
		{"1 burk tomater (300g)", "tomater", floatPtr(1), strPtr("burk")},
		{"mjölk", "mjölk", nil, nil},
		{"halv liter mjölk", "mjölk", floatPtr(0.5), strPtr("liter")},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseIngredientInput(tt.input)
			assert.Equal(t, tt.input, got.OriginalInput, "OriginalInput")
			assert.Equal(t, tt.wantName, got.Name, "Name")
			if tt.wantCnt != nil {
				require.NotNil(t, got.Count, "Count")
				assert.InDelta(t, *tt.wantCnt, *got.Count, 0.001, "Count value")
			} else {
				assert.Nil(t, got.Count, "Count")
			}
			if tt.wantUnit != nil {
				require.NotNil(t, got.Unit, "Unit")
				assert.Equal(t, *tt.wantUnit, *got.Unit, "Unit")
			} else {
				assert.Nil(t, got.Unit, "Unit")
			}
		})
	}
}

func floatPtr(f float64) *float64 { return &f }
func strPtr(s string) *string     { return &s }
