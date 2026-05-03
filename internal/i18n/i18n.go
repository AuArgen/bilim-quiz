package i18n

import (
	"context"
	"encoding/json"
	"os"
	"strings"
)

type Translator struct {
	messages map[string]map[string]string
}

type ctxKey struct{}

var Default *Translator

func Load(dir string) error {
	t := &Translator{messages: make(map[string]map[string]string)}
	langs := []string{"ky", "ru", "en"}
	for _, lang := range langs {
		data, err := os.ReadFile(dir + "/" + lang + ".json")
		if err != nil {
			return err
		}
		m := make(map[string]string)
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		t.messages[lang] = m
	}
	Default = t
	return nil
}

func (t *Translator) T(lang, key string) string {
	if m, ok := t.messages[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if m, ok := t.messages["ky"]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return key
}

func WithLang(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, ctxKey{}, lang)
}

func FromContext(ctx context.Context) string {
	lang, _ := ctx.Value(ctxKey{}).(string)
	if lang == "" {
		return "ky"
	}
	return lang
}

func DetectLang(acceptLang string) string {
	supported := map[string]bool{"ky": true, "ru": true, "en": true}
	for _, part := range strings.Split(acceptLang, ",") {
		tag := strings.TrimSpace(strings.Split(part, ";")[0])
		code := strings.ToLower(strings.Split(tag, "-")[0])
		if supported[code] {
			return code
		}
	}
	return ""
}
