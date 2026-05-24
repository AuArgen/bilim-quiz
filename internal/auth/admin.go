package auth

import (
	"os"
	"strings"
)

func IsAdminEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}

	values := []string{os.Getenv("ADMIN_EMAILS"), os.Getenv("ADMIN_EMAIL")}
	for _, value := range values {
		for _, candidate := range strings.FieldsFunc(value, func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
		}) {
			if strings.ToLower(strings.TrimSpace(candidate)) == email {
				return true
			}
		}
	}

	return false
}
