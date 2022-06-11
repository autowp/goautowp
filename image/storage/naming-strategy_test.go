package storage

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPatternStrategy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Pattern   string
		Result    string
		Extension string
	}{
		{"", "_.jpg", "jpg"}, // "0.jpg"
		{"just.test", "just.test.jpg", "jpg"},
		{"./test/./test/.", "test/test.jpg", "jpg"},
		{"../test/../test/..", "test/test.jpg", "jpg"},
		{"../test////test/..", "test/test.jpg", "jpg"},
	}

	for _, tc := range testCases {
		strategy := NamingStrategyPattern{}
		generated := strategy.Generate(GenerateOptions{
			Pattern:   tc.Pattern,
			Extension: tc.Extension,
		})
		require.EqualValues(t, tc.Result, generated)
	}
}
