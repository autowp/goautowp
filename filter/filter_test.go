package filter

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	testCases := []struct {
		Input  string
		Output string
	}{
		{
			Input:  "абвгдеёжзиклмнопрстуфх ц ч ш щ ъыь эюя",
			Output: "abvgdeiozhziklmnoprstufkh_ts_ch_sh_shch_y_eiuia",
		},
		{
			Input:  "Škoda",
			Output: "skoda",
		},
		{
			Input:  "数据库",
			Output: "shu_ju_ku",
		},
	}

	for _, testCase := range testCases {
		result := SanitizeFilename(testCase.Input)
		require.Equal(t, testCase.Output, result)
	}
}
