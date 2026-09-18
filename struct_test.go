package appgofig

import (
	"reflect"
	"strings"
	"testing"
)

type structTestConfig struct {
	StringValue string  `default:"hello" env:"TEST_STRING"`
	IntValue    int     `default:"42" env:"TEST_INT"`
	FloatValue  float64 `default:"3.14" env:"TEST_FLOAT"`
	BoolValue   bool    `default:"true" env:"TEST_BOOL"`
	Required    string  `default:"value" req:"true"`
	Masked      string  `default:"secret" masked:"true"`
	AliasMask   string  `mask:"true"`
	AliasReq    string  `require:"true"`
}

func TestCheckConfigStruct(t *testing.T) {
	tests := []struct {
		name    string
		cfg     any
		wantErr string
	}{
		{name: "nil", cfg: nil, wantErr: "config cannot be nil"},
		{name: "non-pointer", cfg: structTestConfig{}, wantErr: "config must be a pointer"},
		{name: "nil-pointer", cfg: (*structTestConfig)(nil), wantErr: "config must not be a nil pointer"},
		{name: "pointer-to-non-struct", cfg: new(int), wantErr: "config must point to a struct"},
		{name: "unsupported-type", cfg: &struct{ Values []string }{}, wantErr: "invalid type slice on field Values"},
		{name: "valid", cfg: &structTestConfig{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkConfigStruct(tt.cfg)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("checkConfigStruct() unexpected error: %v", err)
				}
				return
			}

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("checkConfigStruct() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestOnlyContainsSupportedTypes(t *testing.T) {
	if err := onlyContainsSupportedTypes(&structTestConfig{}); err != nil {
		t.Fatalf("unexpected error for supported types: %v", err)
	}

	cfg := &struct {
		Valid   string
		Invalid []byte
	}{}

	err := onlyContainsSupportedTypes(cfg)
	if err == nil || !strings.Contains(err.Error(), "Invalid") {
		t.Fatalf("expected invalid field error, got %v", err)
	}
}

func TestReadConfigDefaults(t *testing.T) {
	entries := readConfigDefaults(&structTestConfig{})

	tests := []struct {
		key          string
		defaultValue string
		valueType    reflect.Kind
		required     bool
		masked       bool
		envKey       string
	}{
		{"StringValue", "hello", reflect.String, false, false, "TEST_STRING"},
		{"IntValue", "42", reflect.Int, false, false, "TEST_INT"},
		{"FloatValue", "3.14", reflect.Float64, false, false, "TEST_FLOAT"},
		{"BoolValue", "true", reflect.Bool, false, false, "TEST_BOOL"},
		{"Required", "value", reflect.String, true, false, ""},
		{"Masked", "secret", reflect.String, false, true, ""},
		{"AliasMask", "", reflect.String, false, true, ""},
		{"AliasReq", "", reflect.String, true, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			entry, ok := entries[tt.key]
			if !ok {
				t.Fatalf("missing entry %q", tt.key)
			}

			if entry.DefaultValue != tt.defaultValue ||
				entry.Value != tt.defaultValue ||
				entry.ValueType != tt.valueType ||
				entry.IsRequired != tt.required ||
				entry.IsMasked != tt.masked ||
				entry.EnvironmentKey != tt.envKey {
				t.Fatalf("entry = %+v, want default=%q type=%v required=%v masked=%v env=%q",
					entry, tt.defaultValue, tt.valueType, tt.required, tt.masked, tt.envKey)
			}
		})
	}
}

func TestCheckRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		entries map[string]*AppConfigEntry
		wantErr bool
	}{
		{
			name: "all present",
			entries: map[string]*AppConfigEntry{
				"Required": {Value: "value", IsRequired: true},
			},
		},
		{
			name: "missing required",
			entries: map[string]*AppConfigEntry{
				"Required": {Value: "", IsRequired: true},
			},
			wantErr: true,
		},
		{
			name: "optional empty",
			entries: map[string]*AppConfigEntry{
				"Optional": {Value: "", IsRequired: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkRequiredFields(tt.entries)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkRequiredFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHasOneBoolTagSet(t *testing.T) {
	field := reflect.TypeOf(struct {
		First   string `first:"true"`
		Second  string `second:"false"`
		Invalid string `invalid:"not-a-bool"`
	}{})

	tests := []struct {
		name string
		idx  int
		tags []string
		want bool
	}{
		{"empty tags", 0, nil, false},
		{"true", 0, []string{"first"}, true},
		{"false", 1, []string{"second"}, false},
		{"invalid", 2, []string{"invalid"}, false},
		{"missing", 0, []string{"missing"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasOneBoolTagSet(field.Field(tt.idx), tt.tags); got != tt.want {
				t.Fatalf("hasOneBoolTagSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadStringFromValue(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{"string", "hello", "hello"},
		{"int", int(42), "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"unsupported", []string{"x"}, " - unsupported type slice - "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readStringFromValue(reflect.ValueOf(tt.val)); got != tt.want {
				t.Fatalf("readStringFromValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyEntryToValue(t *testing.T) {
	t.Run("supported values", func(t *testing.T) {
		cfg := structTestConfig{}
		value := reflect.ValueOf(&cfg).Elem()

		tests := []struct {
			field string
			input string
			want  any
		}{
			{"StringValue", "changed", "changed"},
			{"IntValue", "7", int(7)},
			{"FloatValue", "2.5", float64(2.5)},
			{"BoolValue", "false", false},
			{"IntValue", "0x2a", int(42)},
		}

		for _, tt := range tests {
			t.Run(tt.field+"="+tt.input, func(t *testing.T) {
				field, _ := value.Type().FieldByName(tt.field)
				fieldValue := value.FieldByName(tt.field)

				if err := applyEntryToValue(field, fieldValue, &AppConfigEntry{Value: tt.input}); err != nil {
					t.Fatalf("applyEntryToValue() unexpected error: %v", err)
				}

				if got := fieldValue.Interface(); got != tt.want {
					t.Errorf("%s = %v, want %v", tt.field, got, tt.want)
				}
			})
		}
	})

	t.Run("empty numeric values become zero", func(t *testing.T) {
		cfg := structTestConfig{}
		value := reflect.ValueOf(&cfg).Elem()

		for _, fieldName := range []string{"IntValue", "FloatValue", "BoolValue"} {
			field, _ := value.Type().FieldByName(fieldName)
			fieldValue := value.FieldByName(fieldName)

			if err := applyEntryToValue(field, fieldValue, &AppConfigEntry{}); err != nil {
				t.Fatalf("applyEntryToValue(%s) unexpected error: %v", fieldName, err)
			}

			if !fieldValue.IsZero() {
				t.Errorf("%s = %v, want zero value", fieldName, fieldValue.Interface())
			}
		}
	})

	t.Run("invalid values", func(t *testing.T) {
		cfg := structTestConfig{}
		value := reflect.ValueOf(&cfg).Elem()

		tests := []struct {
			field string
			input string
		}{
			{"BoolValue", "not-bool"},
			{"IntValue", "not-int"},
			{"IntValue", "999999999999999999999999999"},
			{"FloatValue", "not-float"},
		}

		for _, tt := range tests {
			t.Run(tt.field+"="+tt.input, func(t *testing.T) {
				field, _ := value.Type().FieldByName(tt.field)

				if err := applyEntryToValue(
					field,
					value.FieldByName(tt.field),
					&AppConfigEntry{Value: tt.input},
				); err == nil {
					t.Errorf("expected error for %s=%q", tt.field, tt.input)
				}
			})
		}
	})

	t.Run("non-settable field", func(t *testing.T) {
		cfg := structTestConfig{}
		value := reflect.ValueOf(cfg)
		field, _ := value.Type().FieldByName("StringValue")

		err := applyEntryToValue(
			field,
			value.FieldByName("StringValue"),
			&AppConfigEntry{Value: "changed"},
		)
		if err == nil || !strings.Contains(err.Error(), "cannot be set") {
			t.Fatalf("applyEntryToValue() error = %v, want cannot-be-set error", err)
		}
	})

	t.Run("unsupported field type", func(t *testing.T) {
		cfg := struct {
			Values []string
		}{}
		value := reflect.ValueOf(&cfg).Elem()
		field, _ := value.Type().FieldByName("Values")

		err := applyEntryToValue(
			field,
			value.FieldByName("Values"),
			&AppConfigEntry{Value: "ignored"},
		)
		if err == nil || !strings.Contains(err.Error(), "unsupported type slice") {
			t.Fatalf("applyEntryToValue() error = %v, want unsupported-type error", err)
		}
	})
}

func TestGetConfigEntryKeys(t *testing.T) {
	cfg := &struct {
		First  string
		Second int
		Third  bool
	}{}

	want := []string{"First", "Second", "Third"}
	if got := getConfigEntryKeys(cfg); !reflect.DeepEqual(got, want) {
		t.Fatalf("getConfigEntryKeys() = %v, want %v", got, want)
	}
}
