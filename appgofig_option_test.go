package appgofig_test

import (
	"fmt"
	"testing"

	"github.com/smartfactory-kl/appgofig"
)

type ErrorSource struct{}

func (src *ErrorSource) Load(map[string]*appgofig.AppConfigEntry) (map[string]string, error) {
	return nil, fmt.Errorf("test error!")
}

func TestReadConfigRejectsOptionError(t *testing.T) {
	type TestConfig struct {
		Name       string  `default:"app" env:"APP_NAME"`
		Port       int     `default:"8080" env:"APP_PORT"`
		Debug      bool    `default:"false" env:"APP_DEBUG"`
		Secret     string  `default:"secret" env:"APP_SECRET" secret:"true"`
		FloatValue float64 `default:"1"`
	}
	errSource := ErrorSource{}
	_, err := appgofig.ReadConfig(&TestConfig{}, appgofig.WithSources(&errSource))

	if err == nil {
		t.Fatal("ReadCOnfig() expected an error for an option erroring out")
	}
}
