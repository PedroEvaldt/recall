//go:build !integration

package handlers_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/PedroEvaldt/recall/internal/handlers"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "All lower case simple",
			input: "correcttest",
			want:  "correcttest",
		},
		{
			name:  "Upper case simple",
			input: "UpperTest",
			want:  "uppertest",
		},
		{
			name:  "Space case simple",
			input: "space test",
			want:  "space-test",
		},
		{
			name:  "Space Upper case",
			input: "Space Upper Test",
			want:  "space-upper-test",
		},
		{
			name:  "Multiple Spaces case",
			input: "multi   spaces    test",
			want:  "multi-spaces-test",
		},
		{
			name:  "Empty input case",
			input: "",
			want:  "",
		},
		{
			name:  "Hifens case simple",
			input: "hifens-test",
			want:  "hifens-test",
		},
		{
			name:  "Accents pt case",
			input: "Ação e Reação",
			want:  "acao-e-reacao",
		},
		{
			name:  "Ponctuation case",
			input: "Ponctuation?? Case!!",
			want:  "ponctuation-case",
		},
		{
			name:  "Leading/trail case",
			input: "--leading-trail-case---",
			want:  "leading-trail-case",
		},
		{
			name:  "Unicode only case",
			input: "🎉🎉🎉",
			want:  "",
		},
		{
			name:  "Mixed digits case",
			input: "Mixed digits 1.26",
			want:  "mixed-digits-1-26",
		},
		{
			name:  "Long case",
			input: strings.Repeat("a", 200),
			want:  strings.Repeat("a", 80),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handlers.Slugify(tt.input)
			if got != tt.want {
				t.Errorf("expected %s; got %s", tt.want, got)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "Empty case",
			input: "",
			want:  []string{},
		},
		{
			name:  "Simple tag",
			input: "tag1",
			want:  []string{"tag1"},
		},
		{
			name:  "Two simle tags",
			input: "tag1, tag2",
			want:  []string{"tag1", "tag2"},
		},
		{
			name:  "Spaced tags",
			input: "tag1   , tag2  , tag3",
			want:  []string{"tag1", "tag2", "tag3"},
		},
		{
			name:  "Duplicated tags",
			input: "tag1, tag1",
			want:  []string{"tag1"},
		},
		{
			name:  "Multiple commas",
			input: ",,,,,",
			want:  []string{},
		},
		{
			name:  "Mixed case",
			input: "TAG1, tag2    , TAg1,  tag3",
			want:  []string{"tag1", "tag2", "tag3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handlers.NormalizeTags(tt.input)
			if equal := slices.Equal(got, tt.want); !equal {
				t.Errorf("expected %v; got %v", tt.want, got)
			}
		})
	}
}
