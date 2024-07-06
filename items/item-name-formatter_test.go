package items

import (
	"testing"

	"github.com/autowp/goautowp/i18nbundle"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func TestYears(t *testing.T) {
	t.Parallel()

	formatter := ItemNameFormatter{}

	bundle := i18n.NewBundle(language.English)
	localizer := i18n.NewLocalizer(bundle, "en")

	falseVal := false

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
		Today:                  &falseVal,
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

func TestModelYears(t *testing.T) {
	t.Parallel()

	formatter := ItemNameFormatter{}

	bundle, err := i18nbundle.New()
	require.NoError(t, err)

	localizer := bundle.Localizer("en")

	falseVal := false

	itemOptions := ItemNameFormatterOptions{
		BeginModelYear:         1957,
		EndModelYear:           1996,
		BeginModelYearFraction: "½",
		EndModelYearFraction:   "½",
		Spec:                   "Japan",
		SpecFull:               "Japan",
		Body:                   "E39",
		Name:                   "Autobianchi",
		BeginYear:              1957,
		EndYear:                1996,
		Today:                  &falseVal,
		BeginMonth:             3,
		EndMonth:               7,
	}

	result, err := formatter.FormatText(itemOptions, localizer)

	require.NoError(t, err)
	require.Equal(t, "1957½–96½ Autobianchi [Japan] (E39) '03.1957–07.1996", result)

	result, err = formatter.FormatHTML(itemOptions, localizer)

	require.NoError(t, err)
	require.Equal(t, `<span title="model years">1957½–96½</span> `+
		`Autobianchi `+
		`<span class="badge bg-info text-dark" title="Japan" data-toggle="tooltip" data-placement="top">Japan</span> `+
		`(E39)<small> '<span class="realyears" title="years of production"><small class="month">03.</small>1957`+
		`–<small class="month">07.</small>1996</span></small>`, result)
}
