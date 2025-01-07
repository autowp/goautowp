package pictures

import (
	"html"
	"strings"

	"github.com/autowp/goautowp/items"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type PictureNameFormatter struct {
	ItemNameFormatter items.ItemNameFormatter
}

type PictureNameFormatterOptions struct {
	Name  string
	Items []PictureNameFormatterItem
}

type PictureNameFormatterItem struct {
	Item        items.ItemNameFormatterOptions
	Perspective string
}

func (s *PictureNameFormatter) FormatText(
	picture PictureNameFormatterOptions, localizer *i18n.Localizer,
) (string, error) {
	if picture.Name != "" {
		return picture.Name, nil
	}

	if len(picture.Items) > 1 {
		result := make([]string, 0)
		for _, item := range picture.Items {
			result = append(result, item.Item.Name)
		}

		return strings.Join(result, ", "), nil
	} else if len(picture.Items) == 1 {
		item := picture.Items[0]
		prefix := ""

		if item.Perspective != "" {
			translated, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: item.Perspective})
			if err != nil {
				return "", err
			}

			caser := cases.Title(language.English)

			prefix = caser.String(translated) + " "
		}

		formatted, err := s.ItemNameFormatter.FormatText(item.Item, localizer)
		if err != nil {
			return "", err
		}

		return prefix + formatted, nil
	}

	return "Picture", nil
}

func (s *PictureNameFormatter) FormatHTML(
	picture PictureNameFormatterOptions, localizer *i18n.Localizer,
) (string, error) {
	if picture.Name != "" {
		return html.EscapeString(picture.Name), nil
	}

	if len(picture.Items) > 1 {
		result := make([]string, 0)
		for _, item := range picture.Items {
			result = append(result, html.EscapeString(item.Item.Name))
		}

		return strings.Join(result, ", "), nil
	} else if len(picture.Items) == 1 {
		item := picture.Items[0]
		prefix := ""

		if item.Perspective != "" {
			translated, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: item.Perspective})
			if err != nil {
				return "", err
			}

			caser := cases.Title(language.English)

			prefix = html.EscapeString(caser.String(translated)) + " "
		}

		formatted, err := s.ItemNameFormatter.FormatHTML(item.Item, localizer)
		if err != nil {
			return "", err
		}

		return prefix + formatted, nil
	}

	return "Picture", nil
}
