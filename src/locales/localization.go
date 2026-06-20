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
	Logger          *slog.Logger
	Bundle          *i18n.Bundle
	Languages       []language.Tag
	DefaultLanguage language.Tag
}

// Init must be run right after instantiating a new Localization
// Its job is to load the message files that Localizer will use.
func (l *Localization) Init(defaultLanguage language.Tag) {
	l.Bundle = i18n.NewBundle(defaultLanguage)
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

	l.Languages = l.getLanguages()
	l.DefaultLanguage = l.Languages[0]
}

// NewLocalizer creates a new localizer for the given languages.
func (l *Localization) NewLocalizer(languages ...string) *Localizer {
	return &Localizer{
		logger:    l.Logger,
		Localizer: i18n.NewLocalizer(l.Bundle, languages...),
	}
}

func (l *Localization) NewLocalizerAndLanguage(
	polyglotKey string, // make sure this translation key is defined in ALL available languages
	languages ...string,
) (*Localizer, language.Tag) {
	languages = append(languages, l.DefaultLanguage.String())
	localizer := l.NewLocalizer(languages...)

	// Since we use the nice, complex detection of language from i18n, we need to ask the Localizer
	// about the language tags it ended up detecting, especially as it filters by the available ones.
	// This small hack is the only way I've found to get the actual default language of a Localizer.
	_, lang, _ := localizer.Localizer.LocalizeWithTag(
		&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{ID: polyglotKey},
		},
	)

	guessedLanguage := lang
	if lang == language.Und { // Und happens if the polyglotKey is not used in any language
		guessedLanguage = l.DefaultLanguage
	}

	return localizer, guessedLanguage
}

// getLanguages returns the available languages, starting with the default one.
func (l *Localization) getLanguages() []language.Tag {
	languages := make([]language.Tag, 0)
	for _, tag := range l.Bundle.LanguageTags() {
		languages = append(languages, tag)
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
	s, _ := l.Localizer.LocalizeMessage(&i18n.Message{ID: key})
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
