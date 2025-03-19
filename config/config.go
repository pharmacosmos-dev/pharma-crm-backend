package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

type (
	// Config -.
	Config struct {
		App         `yaml:"app"`
		HTTP        `yaml:"http"`
		Log         `yaml:"logger"`
		PG          `yaml:"postgres"`
		Secret      `yaml:"secret"`
		Payment     `yaml:"payment"`
		Integration `yaml:"integration"`
	}

	// App -.
	App struct {
		Name    string `env-required:"true" yaml:"name"    env:"APP_NAME"`
		Version string `env-required:"true" yaml:"version" env:"APP_VERSION"`
	}

	// HTTP -.
	HTTP struct {
		Port string `env-required:"true" yaml:"port" env:"HTTP_PORT"`
	}

	// Log -.
	Log struct {
		Level string `env-required:"true" yaml:"log_level"   env:"LOG_LEVEL"`
	}
	// Token Secret Key -.
	Secret struct {
		SecretKey           string `env-required:"true" yaml:"log_level"   env:"SECRET_KEY"`
		HeshKey             string `env-required:"true" yaml:"log_level"   env:"HESH_KEY"`
		Password1C          string `env-required:"true" yaml:"log_level"   env:"PASSWORD_1C"`
		ExternalAPIUsername string `env-required:"true" yaml:"log_level"   env:"EXTERNAL_API_USERNAME"`
		ExternalAPIPassword string `env-required:"true" yaml:"log_level"   env:"EXTERNAL_API_PASSWORD"`
	}

	// PG -.
	PG struct {
		PoolMax int    `env-required:"true" yaml:"pool_max" env:"PG_POOL_MAX"`
		URL     string `env-required:"true" yaml:"pg_url" env:"PG_URL"`
		DbHost  string `env-required:"true" yaml:"pg_host" env:"PG_HOST"`
		DbPort  string `env-required:"true" yaml:"pg_port" env:"PG_PORT"`
		DbUser  string `env-required:"true" yaml:"pg_user" env:"PG_USER"`
		DbPass  string `env-required:"true" yaml:"pg_pass" env:"PG_PASS"`
		DbName  string `env-required:"true" yaml:"pg_db" env:"PG_DB"`
	}
	Payment struct {
		ClickEndpointUrl   string `env-required:"true" yaml:"click_endpoint_url" env:"CLICK_ENDPOINT_URL"`
		UzumEndpointUrl    string `env-required:"true" yaml:"uzum_endpoint_url" env:"UZUM_ENDPOINT_URL"`
		PaymeGoEndpointUrl string `env-required:"true" yaml:"payme_go_endpoint_url" env:"PAYME_GO_ENDPOINT_URL"`
	}
	Integration struct {
		BaseUrl1C        string `env-required:"true" yaml:"base_url_1c" env:"BASE_URL_1C"`
		BaseUsername1C   string `env-required:"true" yaml:"base_username_1c" env:"BASE_USERNAME_1C"`
		BasePassword1C   string `env-required:"true" yaml:"base_password_1c" env:"BASE_PASSWORD_1C"`
		SoliqIkpuBaseUrl string `env-required:"true" yaml:"soliq_ikpu_base_url" env:"SOLIQ_IKPU_BASE_URL"`
	}
)

// NewConfig returns app config.
func Load() Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Failed to read env: %v", err.Error())
	}
	c := Config{}
	c.App.Name = cast.ToString(GetOrReturnDefaultValue("APP_NAME", "pharma_backend"))
	c.App.Version = cast.ToString(GetOrReturnDefaultValue("APP_VERSION", "1.0.0"))
	c.HTTP.Port = cast.ToString(GetOrReturnDefaultValue("HTTP_PORT", "8080"))
	c.Log.Level = cast.ToString(GetOrReturnDefaultValue("LOG_LEVEL", "debug"))

	c.PG.DbHost = cast.ToString(GetOrReturnDefaultValue("PG_HOST", "localhost"))
	c.PG.DbPort = cast.ToString(GetOrReturnDefaultValue("PG_PORT", "5432"))
	c.PG.DbUser = cast.ToString(GetOrReturnDefaultValue("PG_USER", "username"))
	c.PG.DbPass = cast.ToString(GetOrReturnDefaultValue("PG_PASS", "password"))
	c.PG.DbName = cast.ToString(GetOrReturnDefaultValue("PG_DB", "dbname"))
	c.PG.PoolMax = cast.ToInt(GetOrReturnDefaultValue("PG_POOL_MAX", 2))
	c.Secret.SecretKey = cast.ToString(GetOrReturnDefaultValue("SECRET_KEY", "6fb5619d-8c30-4e85-a1e3-3f4d142498a0"))
	c.Secret.HeshKey = cast.ToString(GetOrReturnDefaultValue("HESH_KEY", "6fb5619d-8c30-4e85-a1e3-3f4d142498a0"))
	c.Secret.Password1C = cast.ToString(GetOrReturnDefaultValue("PASSWORD_1C", "6fb5619d-8c30-4e85-a1e3-3f4d142498a0"))
	c.Secret.ExternalAPIUsername = cast.ToString(GetOrReturnDefaultValue("EXTERNAL_API_USERNAME", "pharmaexternalapis"))
	c.Secret.ExternalAPIPassword = cast.ToString(GetOrReturnDefaultValue("EXTERNAL_API_PASSWORD", "lai3lahxoPo{aph9"))
	c.Payment.ClickEndpointUrl = cast.ToString(GetOrReturnDefaultValue("CLICK_ENDPOINT_URL", "http://localhost:8080"))
	c.Payment.PaymeGoEndpointUrl = cast.ToString(GetOrReturnDefaultValue("PAYME_GO_ENDPOINT_URL", "http://localhost:8080"))
	c.Payment.UzumEndpointUrl = cast.ToString(GetOrReturnDefaultValue("UZUM_ENDPOINT_URL", "http://localhost:8080"))
	c.Integration.BaseUrl1C = cast.ToString(GetOrReturnDefaultValue("BASE_URL_1C", "http://localhost:8080"))
	c.Integration.BaseUsername1C = cast.ToString(GetOrReturnDefaultValue("BASE_USERNAME_1C", "pharma"))
	c.Integration.BasePassword1C = cast.ToString(GetOrReturnDefaultValue("BASE_PASSWORD_1C", "password"))
	c.Integration.SoliqIkpuBaseUrl = cast.ToString(GetOrReturnDefaultValue("SOLIQ_IKPU_BASE_URL", "http://localhost:8080"))
	return c
}

func GetOrReturnDefaultValue(key string, defaultValue interface{}) interface{} {
	val, exists := os.LookupEnv(key)
	if exists {
		return val
	}

	return defaultValue
}
