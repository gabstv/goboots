package i18n

type Language interface {
	Meta(key string) string
	Ctx(ctx string) func(id string, v ...interface{}) string
	T(id string, v ...interface{}) string
}

type Provider interface {
	L(code string) Language
	LanguageCodes() []string
}
