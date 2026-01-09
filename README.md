# AppGofig (AppConfig for Go)

![Coverage](https://codecov.io/gh/kzoltner/appgofig/branch/main/graph/badge.svg)

Using a struct (and one description map) as single source of truth to add configuration to Go applications.

> [!note]
> This is a very simplistic approach to adding simple key:value pair style configuration to your applications. Nothing more, nothing less.

# Install

Should be a simple go get command:

```bash
go get github.com/kzoltner/appgofig
```

# How to use

First, create your Config and your ConfigDescriptions (optional, only needed for documentation):

```go
type Config struct {
	MyOwnSetting    int64  `default:"10" env:"MY_OWN_SETTING"`
	MyStringSetting string `default:"hello" env:"MY_STRING_SETTING" req:"true"`
}

var configDescriptions map[string]string = map[string]string{
	"MyOwnSetting":    "This is just a simple example description so this map is not empty",
	"MyStringSetting": "This is just a string setting that is empty but required.",
}
```

Next, instantiate your config and then read the config values by calling `ReadConfig()`:

```go
cfg := &Config{}

if err := appgofig.ReadConfig(cfg); err != nil {
	log.Fatal(err)
}

appgofig.LogConfig(cfg, os.Stdout)
```

> [!caution]
> Make sure to use a pointer to your struct, not the struct itself.

Now, using your config should be as easy as accessing the struct itself:

```go
log.Println(cfg.MyOwnSetting)
```

## The `Config` struct

The `Config` struct determines your whole configuration. You can name it whatever you want.
The following tags are usable:

| Tag Name  | Content                                                                                                                   |
| --------- | ------------------------------------------------------------------------------------------------------------------------- |
| `env`     | Key used for Environment Variables. If this is empty, it defaults to the field name                                       |
| `default` | String representation of a default value. Otherwise an empty string is used.                                              |
| `req`     | If set to "true", this config setting cannot be empty. Only applies to string values and is ignored on non-string values. |
| `mask`    | If set to "true", this will mask the value of a field when using `LogConfig()`                                            |

Example entry:

```go
type Config struct {
	MyOwnSetting string `env:"ENV_MY_OWN_SETTING" default:"myDefaultValue" req:"true"`
}
```

> [!important]
> Due to my own needs, only four types are allowed: `string`, `int64`, `float64` and `bool`.

## Available Options

The `ReadConfig()` method has a second parameter for `With...()` option functions.
The following are available:

- `WithReadMode(readMode ConfigReadMode)` to set a read mode
- `WithYamlFile(filePath string)` to set a specific YAML file
- `WithNewDefaults(newDefaults map[string]string)` to set new defaults (e.g. for testing)

Check the `example` folder on how to use them.

### ReadModes

There are four read modes available:

| ReadMode                       | Description                                                          |
| ------------------------------ | -------------------------------------------------------------------- |
| `appgofig.ReadModeEnvOnly`     | Only uses Environment to read values                                 |
| `appgofig.ReadModeYamlOnly`    | Only uses a YAML file                                                |
| `appgofig.ReadModeEnvThenYaml` | First read env, then apply YAML (overwriting env values if present)  |
| `appgofig.ReadModeYamlThenEnv` | First read YAML, then apply env (overwriting YAML values if present) |

### Using yaml files

When no YAML file is specified using `WithYamlFile()`, but a YAML ReadMode is used, this list
of paths is used to look for YAML files. First hit is used:

```go
defaultYamlPaths := []string{"config.yml", "config.yaml", "config/config.yml", "config/config.yaml"}
```

> [!important]
> To keep it simple, only flat key:value pair YAMLs are allowed. No nesting should be there.

# Documentation

Two methods are provided to automatically create documentation about your configuration.
Check the `example` folder for how they could look like.

### Markdown

Using `WriteToMarkdownFile()` you can generate a markdown file containing a simple table.

```go
if err := appgofig.WriteToMarkdownFile(cfg, configDescriptions, "example/MarkdownExample.md"); err != nil {
	log.Fatal(err)
}
```

### Example config.yaml

Similarly, using `WriteToYamlExampleFile()` will get you a config.yml example with comments explaining each entry

```go
if err := appgofig.WriteToYamlExampleFile(cfg, configDescriptions, "example/ConfigYamlExample.yaml"); err != nil {
	log.Fatal(err)
}
```

# Tests
A basic set of tests is included. To run: 

```go
go test -cover ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

You can admire the coverage in a browser then.
