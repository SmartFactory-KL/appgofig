package appgofig

import "testing"

func TestGetEnvKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{
			name:   "prefix and key",
			prefix: "app",
			key:    "database_name",
			want:   "APP_DATABASE_NAME",
		},
		{
			name:   "empty prefix",
			prefix: "",
			key:    "database_name",
			want:   "DATABASE_NAME",
		},
		{
			name:   "empty key",
			prefix: "app",
			key:    "",
			want:   "APP_",
		},
		{
			name:   "empty prefix and key",
			prefix: "",
			key:    "",
			want:   "",
		},
		{
			name:   "prefix already has trailing underscore",
			prefix: "APP_",
			key:    "DATABASE_NAME",
			want:   "APP_DATABASE_NAME",
		},
		{
			name:   "multiple trailing underscores on prefix",
			prefix: "APP___",
			key:    "DATABASE",
			want:   "APP___DATABASE",
		},
		{
			name:   "key starts with underscore",
			prefix: "APP",
			key:    "_DATABASE",
			want:   "APP_DATABASE",
		},
		{
			name:   "prefix and key already uppercase",
			prefix: "APP",
			key:    "DATABASE_NAME",
			want:   "APP_DATABASE_NAME",
		},
		{
			name:   "mixed case",
			prefix: "myApp",
			key:    "databaseName",
			want:   "MYAPP_DATABASE_NAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getEnvKey(tt.prefix, tt.key)

			if got != tt.want {
				t.Errorf("getEnvKey(%q, %q) = %q, want %q",
					tt.prefix, tt.key, got, tt.want)
			}
		})
	}
}

func TestToUpperSnakeCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple camel case",
			input: "appName",
			want:  "APP_NAME",
		},
		{
			name:  "single word",
			input: "database",
			want:  "DATABASE",
		},
		{
			name:  "already uppercase",
			input: "DATABASE",
			want:  "DATABASE",
		},
		{
			name:  "already snake case",
			input: "database_name",
			want:  "DATABASE_NAME",
		},
		{
			name:  "already upper snake case",
			input: "DATABASE_NAME",
			want:  "DATABASE_NAME",
		},
		{
			name:  "pascal case",
			input: "DatabaseName",
			want:  "DATABASE_NAME",
		},
		{
			name:  "multiple words",
			input: "DatabaseConnectionTimeout",
			want:  "DATABASE_CONNECTION_TIMEOUT",
		},
		{
			name:  "acronym followed by word",
			input: "HTTPPort",
			want:  "HTTP_PORT",
		},
		{
			name:  "word followed by acronym",
			input: "DatabaseURL",
			want:  "DATABASE_URL",
		},
		{
			name:  "acronym followed by acronym",
			input: "HTTPURL",
			want:  "HTTPURL",
		},
		{
			name:  "single character",
			input: "A",
			want:  "A",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "leading underscore",
			input: "_databaseName",
			want:  "_DATABASE_NAME",
		},
		{
			name:  "trailing underscore",
			input: "databaseName_",
			want:  "DATABASE_NAME_",
		},
		{
			name:  "multiple underscores",
			input: "database__name",
			want:  "DATABASE__NAME",
		},
		{
			name:  "numbers",
			input: "http2Port",
			want:  "HTTP2_PORT",
		},
		{
			name:  "number followed by uppercase",
			input: "version2Config",
			want:  "VERSION2_CONFIG",
		},
		{
			name:  "unicode lowercase and uppercase",
			input: "überName",
			want:  "ÜBER_NAME",
		},
		{
			name:  "unicode uppercase",
			input: "ÜberName",
			want:  "ÜBER_NAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := toUpperSnakeCase(tt.input)

			if got != tt.want {
				t.Errorf("toUpperSnakeCase(%q) = %q, want %q",
					tt.input, got, tt.want)
			}
		})
	}
}
