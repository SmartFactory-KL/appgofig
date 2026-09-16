package appgofig

// AppConfigEntry represents the entry within a configuration struct including the possible tag values
type AppConfigEntry struct {
	// The key, being the actual field name of the struct entry
	Key string

	// current string value, as read from the sources
	Value string

	// the default value as taken from the config struct, defaults to empty string
	DefaultValue string

	// If true, this needs len(Value) > 0
	IsRequired bool

	// If true, Value will be masked when using VisitConfigValues
	IsMasked bool

	// If non-nil, this will take precedence over the env key that would generate from the Key. Prefix will still be added to it though.
	// use this for cases like HTTPURL which would becomd HTTP_URL
	EnvironmentKey string
}

// AppGofigOptions contains the config for the configuration reader
type AppGofigOptions struct {
	// list of sources to read config from, in order
	Sources []AppGofigSource

	// overrides to apply at the very end, overwriting any other values
	Overrides map[string]string
}

// single option interface
type AppGofigOption func(*AppGofigOptions)

// WithSources will apply sources in the given order, overwriting previous values and starting from default
func WithSources(sources []AppGofigSource) AppGofigOption {
	return func(options *AppGofigOptions) {
		options.Sources = sources
	}
}

// WithOverrides will set values after all other sources have been applied
func WithOverrides(values map[string]string) AppGofigOption {
	return func(options *AppGofigOptions) {
		options.Overrides = values
	}
}
