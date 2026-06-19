package locales

import (
	"embed"
	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"log/slog"
)

//go:embed locale.*.toml
var LocaleFS embed.FS

// Localization is our main localization service.
// It basically acts as a factory for Localizer.
type Localization struct {
	Logger *slog.Logger
	Bundle *i18n.Bundle
}

// Init must be run right after instantiating a new Localization
// Its job is to load the message files that Localizer will use.
func (l *Localization) Init() {
	l.Bundle = i18n.NewBundle(language.English)
	l.Bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	dirEntries, err := LocaleFS.ReadDir(".")
	if err != nil {
		panic(err)
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}

		_, err = l.Bundle.LoadMessageFileFS(LocaleFS, dirEntry.Name())
		if err != nil {
			panic(err)
		}
	}
}

func (l *Localization) GetLocalizer(languages ...string) *Localizer {
	return &Localizer{
		logger:    l.Logger,
		Localizer: i18n.NewLocalizer(l.Bundle, languages...),
	}
}

// GetLanguages returns the available languages, starting with the default one.
func (l *Localization) GetLanguages() []string {
	languages := make([]string, 0)
	for _, tag := range l.Bundle.LanguageTags() {
		languages = append(languages, tag.String())
	}
	return languages
}

// Localizer is our sugary wrapper around the localize methods of i18n
type Localizer struct {
	logger    *slog.Logger
	Localizer *i18n.Localizer
}

// T translates the message identified by its key
func (l *Localizer) T(key string) string {
	s, err := l.Localizer.LocalizeMessage(&i18n.Message{ID: key})
	if err != nil {
		panic(err)
	}
	return s
}

// Tf translates and formats the message identified by its key
func (l *Localizer) Tf(key string, data map[string]interface{}) string {
	s, _ := l.Localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{ID: key},
		TemplateData:   data,
	})
	return s
}

// Tp translates and pluralizes the message identified by its key
func (l *Localizer) Tp(
	key string,
	amount interface{},
) string {
	s, _ := l.Localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{ID: key},
		PluralCount:    amount,
	})
	return s
}

// Tfp translates, formats and pluralizes the message identified by its key
func (l *Localizer) Tfp(
	key string,
	data map[string]interface{},
	amount interface{},
) string {
	s, err := l.Localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{ID: key},
		TemplateData:   data,
		PluralCount:    amount,
	})
	if err != nil {
		l.logger.Warn("failed to localize", "err", err)
	}
	return s
}
