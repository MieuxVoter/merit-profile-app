package locales

import (
	"github.com/tyler-sommer/stick"
)

type LocalizationExtension struct {
	Localization *Localization
	Localizers   map[string]*Localizer // memoization of localizers
}

func (l LocalizationExtension) Init(e *stick.Env) error {
	e.Filters["t"] = FilterTranslateFactory(l)
	return nil
}

func (l LocalizationExtension) FindLocalizer(ctx stick.Context) *Localizer {
	language, languageFound := ctx.Scope().Get("language")
	if !languageFound {
		language = stick.Value("en")
	}

	localizer, localizerFound := l.Localizers[language.(string)]
	if !localizerFound {
		localizer = l.Localization.GetLocalizer(language.(string), "en")
		l.Localizers[language.(string)] = localizer
	}

	return localizer
}

func FilterTranslateFactory(
	l LocalizationExtension,
) func(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	return func(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
		return stick.Value(l.FindLocalizer(ctx).T(val.(string)))
	}
}
