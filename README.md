[![codecov](https://codecov.io/github/smartfactory-kl/appgofig/branch/main/graph/badge.svg?token=JORBSOKMZX)](https://codecov.io/github/smartfactory-kl/appgofig)

# AppGofig (AppConfig for Go)

Using a struct (and one optional description map) as single source of truth to add configuration to Go applications.
The approach is simplistic on purpose, supporting only flat configuration structures and only four field types: `string`, `int`, `float64`, `bool`.
If you need more features there are probably dozens of proper configuration libraries for Go.

# Install

```bash
go get github.com/smartfactory-kl/appgofig
```

# Quick Start

Basic Example:

```go
package main

import (
	"log"

	"github.com/smartfactory-kl/appgofig"
)

// Define the Config struct itself
type Config struct {
	AppName string  `default:"my-app"`
	Port    int     `default:"8080"`
	Debug   bool    `default:"false"`
	Ratio   float64 `default:"1.0"`
}

func main() {
	// Read config using the struct instance
	// By default (with no sources applied) it will simply use the specified default values
	cfg, err := appgofig.ReadConfig[Config]()

	if err != nil {
		log.Fatal(err)
	}

	// Config options are autocompleted
	log.Println(cfg.AppName, cfg.Port, cfg.Debug, cfg.Ratio)

	// Usually, at least one source should be used, for example the environment
	cfgFromEnv, err := appgofig.ReadConfig[Config](
		appgofig.WithSources(
			appgofig.EnvironmentSource(),
		),
	)

	if err != nil {
		log.Fatal(err)
	}

	// Config options are still autocompleted
	log.Println(cfgFromEnv.AppName, cfgFromEnv.Port, cfgFromEnv.Debug, cfgFromEnv.Ratio)
}
```

## Struct Tags

Most metadata is set using struct tags.

### `default`

Sets the initial value for a field. Needs to be the string representation.

```
Port int `default:"8080"`
```

### `env`

Sets the Environment Key to look for. If omitted, the field name is converted to UPPER_SNAKE_CASE and used as Environment Key.

```go
type Config struct {
	AppName string `env:"APPLICATION_NAME"` // will be looking for APPLICATION_NAME
	HTTPPort int // will be looking for HTTP_PORT
}
```

> [!note]
> Note that an optional Environment Prefix will still be added to both variants

### `required`, `require`, `req`

Setting this to `true` requires a field to not be empty before type conversion.

> [!note]
> Empty values for booleans are considered to be false

```go
APIKey string `required:"true"`
```

### `masked`, `mask`

Setting this to `true` masks the value and the default value of the field when using `VisitConfigEntries` and in any generated documentation.

```go
APIKey string `masked:"true"`
```

> [!note]
> The value will be replaced with `[Masked (len: x)]` with `x` being the byte length of the (stringified) value or default value

## Configuration Sources

Specifying no sources will result in a Config with only default values.
To specify a source, use the `WithSources()` option.

### Environment as source

Two variants are available:

```go
// Using no Prefix
appgofig.WithSources(
	appgofig.EnvironmentSource()
)

// Using a Prefix on all Keys
appgofig.WithSources(
	appgofig.PrefixedEnvironmentSource("MY_PREFIX")
)
```

For a field tagged with `env:"APPLICATION_NAME"` this would result in the following keys to look up:

- EnvironmentSource -> `APPLICATION_NAME`
- PrefixedEnvironmentSource -> `MY_PREFIX_APPLICATION_NAME`

### YAML file as source

The YAMLSource can either use predefined default file paths or a single specified one:

```go
// Using the first match of the defaults
appgofig.WithSources(
	appgofig.YAMLSource()
)

// Using a single file specified by its path
appgofig.WithSources(
	appgofig.SpecificYAMLSource("config.dev.yml")
)
```

The default file paths are (in order)

- `config.yml`
- `config.yaml`
- `config/config.yml`
- `config/config.yaml`

YAML Keys must match the Go struct field name exactly and it must contain a flat hierarchy. Nested objects are not supported.

```yaml
AppName: yaml-app
Port: 9000
Debug: true
```

## Overrides

Another option is `WithOverrides`, containing a `map[string]string` that will always be applied last.

```go
appgofig.WithOverrides(map[string]string{
	"AppVersion": "1.0.0-rc4",
})
```

## Combining sources

Sources and Overrides can be combined. Sources will be applied in order, Overrides always at the end.
If multiple sources define the same key, later sources will overwrite earlier ones.

```go
cfg, err := appgofig.ReadConfig[Config](
	appgofig.WithSources(
		appgofig.YAMLSource(),
		appgofig.PrefixedEnvironmentSource("APP"),
	),
	appgofig.WithOverrides(map[string]string{
		"Port": "7000",
	}),
)
```

## Inspecting configuration values

Using `VisitConfigEntries`, all configuration values can be inspected:

```go
err := appgofig.VisitConfigEntries(
	&cfg,
	func(entry appgofig.AppConfigEntry) {
		log.Printf("%s=%s", entry.Key, entry.Value)
	},
)
```

> [!note]
> Masked values will be reported as `[Masked (len: N)]`

AppConfigEntry includes:

```go
type AppConfigEntry struct {
	Key            string
	Value          string
	ValueType      reflect.Kind
	DefaultValue   string
	IsRequired     bool
	IsMasked       bool
	EnvironmentKey string
}
```

## Generating Documentation

```go
descriptions := map[string]string{
	"AppName": "The application name.",
	"Port":    "The listening port.",
}

if err := appgofig.CreateConfigDocumentation(
	&cfg,
	descriptions,
	"ENV_PREFIX",
	"docs",
); err != nil {
	log.Fatal(err)
}
```

This will generate:

- `docs/DefaultDocumentation.md`
- `docs/config.example.yaml`

The Markdown document contains a configuration overview, Docker Compose example, and Docker run example.

They can also be created individually:

```go
err := appgofig.CreateConfigMarkdownDocument(
	&cfg,
	descriptions,
	"ENV_PREFIX",
	"docs/config.md",
)

err := appgofig.CreateConfigExampleYAML(
	&cfg,
	descriptions,
	"ENV_PREFIX",
	"docs/config.example.yaml",
)
```

> [!note]
> `ENV_PREFIX` should only be used if one of the sources is the prefixed environment. Otherwise simply use `""`

## Custom Sources

Custom sources implement the `AppGofigSource` interface:

```go
type AppGofigSource interface {
	Load(map[string]*appgofig.AppConfigEntry) (map[string]string, error)
}
```

The returned map should use configuration struct field names as keys. Unknown keys are ignored.
The input map parameter is for informational purposes only and should never be altered (e.g. reading the EnvironmentKey or its ValueType)

## Testing

Run tests:

```bash
go test ./... -cover
```

Create coverage HTML report:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```
