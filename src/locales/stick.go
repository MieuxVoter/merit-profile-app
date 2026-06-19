package locales

import (
	"github.com/tyler-sommer/stick"
)

type LocalizationExtension struct {
	Localization *Localization
}

func (l LocalizationExtension) Init(e *stick.Env) error {
	e.Filters["t"] = FilterTranslateFactory(l)
	return nil
}

func FilterTranslateFactory(l LocalizationExtension) func(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	return func(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
		language, found := ctx.Scope().Get("language")
		if !found {
			language = stick.Value("en")
		}
		localizer := l.Localization.GetLocalizer(language.(string))
		return stick.Value(localizer.T(val.(string)))
	}
}
