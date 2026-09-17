package appgofig

import (
	"strings"
	"unicode"
)

// getEnvKey returns UPPER_CASE_SNAKE version while joining prefix and key with underscore
func getEnvKey(prefix string, key string) string {
	sb := strings.Builder{}

	if len(prefix) > 0 {
		sb.WriteString(strings.TrimSuffix(strings.ToUpper(prefix), "_"))
		sb.WriteRune('_')
	}

	sb.WriteString(strings.TrimPrefix(strings.ToUpper(toUpperSnakeCase(key)), "_"))

	return sb.String()
}

// toUpperSnakeCase will convert camelCase or PascalCase to UPPER_SNAKE_CASE
func toUpperSnakeCase(input string) string {
	var sb strings.Builder

	runes := []rune(input)

	for idx, char := range runes {
		if unicode.IsUpper(char) {
			// if it is upper case, check before and after for lowercase
			if idx > 0 && (unicode.IsLower(runes[idx-1]) || (idx+1) < len(runes) && unicode.IsLower(runes[idx+1])) {
				sb.WriteByte('_')
			}

		}

		// write the char itself
		sb.WriteRune(unicode.ToUpper(char))
	}

	return sb.String()
}

// escapeMarkdown escapes characters that can interfere with Markdown tables.
func escapeMarkdown(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "|", `\|`)
	value = strings.ReplaceAll(value, "\n", " ")

	return value
}

// shellQuote quotes a string for use in a shell command.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
