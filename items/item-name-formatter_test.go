package items

import (
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func TestYears(t *testing.T) {
	t.Parallel()

	formatter := ItemNameFormatter{}

	bundle := i18n.NewBundle(language.English)
	localizer := i18n.NewLocalizer(bundle, "en")

	f := false

	itemOptions := ItemNameFormatterOptions{
		BeginModelYear:         0,
		EndModelYear:           0,
		BeginModelYearFraction: "",
		EndModelYearFraction:   "",
		Spec:                   "",
		SpecFull:               "",
		Body:                   "",
		Name:                   "Autobianchi",
		BeginYear:              1957,
		EndYear:                1996,
		Today:                  &f,
		BeginMonth:             0,
		EndMonth:               0,
	}

	result, err := formatter.FormatText(itemOptions, localizer)

	require.NoError(t, err)
	require.Equal(t, "Autobianchi '1957–96", result)

	result, err = formatter.FormatHTML(itemOptions, localizer)

	require.NoError(t, err)
	require.Equal(t, "Autobianchi '1957–96", result)
}
