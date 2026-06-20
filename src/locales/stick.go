package locales

import (
	"fmt"
	"github.com/tyler-sommer/stick"
)

// LocalizationExtension implements stick.Extension to add localization utilities
type LocalizationExtension struct {
	Localization *Localization
	Localizers   map[string]*Localizer // memoization of localizers
}

// Init is the main API of stick.Extension
func (l LocalizationExtension) Init(e *stick.Env) error {
	e.Filters["t"] = FilterTranslateFactory(l)
	e.Filters["trans"] = FilterTranslateFactory(l)
	return nil
}

// FindLocalizer helps finding a localizer for a language, using memoization
func (l LocalizationExtension) FindLocalizer(ctx stick.Context) *Localizer {
	languageValue, languageFound := ctx.Scope().Get("language")
	language := languageValue.(string)
	if !languageFound {
		language = l.Localization.DefaultLanguage.String()
	}

	localizer, localizerFound := l.Localizers[language]
	if !localizerFound {
		localizer = l.Localization.NewLocalizer(
			language,
			l.Localization.DefaultLanguage.String(),
		)
		l.Localizers[language] = localizer
	}

	return localizer
}

// FilterTranslateFactory allows our filter function to access the Localization instance.
func FilterTranslateFactory(
	l LocalizationExtension,
) func(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	return func(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
		if len(args) > 0 {
			data := make(map[string]interface{})
			for i, arg := range args {
				data[fmt.Sprintf("Value%d", i)] = arg
			}
			return stick.Value(l.FindLocalizer(ctx).Tf(val.(string), data))
		}
		return stick.Value(l.FindLocalizer(ctx).T(val.(string)))
	}
}
