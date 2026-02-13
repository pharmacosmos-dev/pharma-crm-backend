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
		PG          `yaml:"postgres"`
		Secret      `yaml:"secret"`
		Integration `yaml:"integration"`
	}

	// App -.
	App struct {
		Name    string `env-required:"true" yaml:"name"    env:"APP_NAME"`
		Version string `env-required:"true" yaml:"version" env:"APP_VERSION"`
		Port    string `env-required:"true" yaml:"port" env:"HTTP_PORT"`
		Level   string `env-required:"true" yaml:"log_level"   env:"LOG_LEVEL"`
	}
	// Token Secret Key -.
	Secret struct {
		SecretKey           string `env-required:"true" yaml:"log_level"   env:"SECRET_KEY"`
		HashKey             string `env-required:"true" yaml:"log_level"   env:"HASH_KEY"`
		OnecPassword        string `env-required:"true" yaml:"log_level"   env:"ONEC_PASSWORD"`
		ExternalApiUsername string `env-required:"true" yaml:"log_level"   env:"EXTERNAL_API_USERNAME"`
		ExternalApiPassword string `env-required:"true" yaml:"log_level"   env:"EXTERNAL_API_PASSWORD"`
		FileBaseURL         string `env-required:"true" yaml:"file_base_url"   env:"FILE_BASE_URL"`
		UzumClientId        string `env-required:"false" yaml:"uzum_client_id"   env:"UZUM_CLIENT_ID"`
		UzumClientSecret    string `env-required:"false" yaml:"uzum_client_secret"   env:"UZUM_CLIENT_SECRET"`
		OAuthTokenExpiry    int    `env-required:"false" yaml:"oauth_token_expiry"   env:"OAUTH_TOKEN_EXPIRY"`
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
	// Integration -.
	Integration struct {
		ClickApiUrl      string `env-required:"true" yaml:"click_api_url" env:"CLICK_API_URL"`
		UzumApiUrl       string `env-required:"true" yaml:"uzum_api_url" env:"UZUM_API_URL"`
		PaymeApiUrl      string `env-required:"true" yaml:"payme_api_url" env:"PAYME_API_URL"`
		AlifApiUrl       string `env-required:"true" yaml:"alif_api_url" env:"ALIF_API_URL"`
		AlifToken        string `env-required:"true" yaml:"alif_token"   env:"ALIF_TOKEN"`
		OnecApiUrl       string `env-required:"true" yaml:"onec_api_url" env:"ONEC_API_URL"`
		OnecApiUsername  string `env-required:"true" yaml:"onec_api_username" env:"ONEC_API_USERNAME"`
		OnecApiPassword  string `env-required:"true" yaml:"onec_api_password" env:"ONEC_API_PASSWORD"`
		TasnifApiUrl     string `env-required:"true" yaml:"tasnif_api_url" env:"TASNIF_API_URL"`
		NoorApiUrl       string `env-required:"true" yaml:"noor_api_url" env:"NOOR_API_URL"`
		NoorApiToken     string `env-required:"true" yaml:"noor_api_token" env:"NOOR_API_TOKEN"`
		DmedApiUrl       string `env-required:"true" yaml:"dmed_api_url" env:"DMED_API_URL"`
		DmedApiToken     string `env-required:"true" yaml:"dmed_api_token" env:"DMED_API_TOKEN"`
		OsonAptekaApiUrl string `env-required:"true" yaml:"oson_apteka_api_url" env:"OSON_APTEKA_API_URL"`
	}
)

// NewConfig returns app config.
func Load() Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("could not read .env: %v", err)
	}
	c := Config{}
	c.App.Name = cast.ToString(GetOrReturnDefaultValue("APP_NAME", "pharma_backend"))
	c.App.Version = cast.ToString(GetOrReturnDefaultValue("APP_VERSION", "1.0.0"))
	c.App.Port = cast.ToString(GetOrReturnDefaultValue("HTTP_PORT", "8080"))
	c.App.Level = cast.ToString(GetOrReturnDefaultValue("LOG_LEVEL", "debug"))

	c.PG.DbHost = cast.ToString(GetOrReturnDefaultValue("PG_HOST", "localhost"))
	c.PG.DbPort = cast.ToString(GetOrReturnDefaultValue("PG_PORT", "5432"))
	c.PG.DbUser = cast.ToString(GetOrReturnDefaultValue("PG_USER", "username"))
	c.PG.DbPass = cast.ToString(GetOrReturnDefaultValue("PG_PASS", "password"))
	c.PG.DbName = cast.ToString(GetOrReturnDefaultValue("PG_DB", "dbname"))
	c.PG.PoolMax = cast.ToInt(GetOrReturnDefaultValue("PG_POOL_MAX", 2))

	c.Secret.SecretKey = cast.ToString(GetOrReturnDefaultValue("SECRET_KEY", "6fb5619d-8c30-4e85-a1e3-3f4d142498a0"))
	c.Secret.HashKey = cast.ToString(GetOrReturnDefaultValue("HASH_KEY", "6fb5619d-8c30-4e85-a1e3-3f4d142498a0"))
	c.Secret.OnecPassword = cast.ToString(GetOrReturnDefaultValue("ONEC_PASSWORD", "6fb5619d-8c30-4e85-a1e3-3f4d142498a0"))
	c.Secret.ExternalApiUsername = cast.ToString(GetOrReturnDefaultValue("EXTERNAL_API_USERNAME", "pharmaexternalapis"))
	c.Secret.ExternalApiPassword = cast.ToString(GetOrReturnDefaultValue("EXTERNAL_API_PASSWORD", "lai3lahxoPo{aph9"))
	c.Secret.FileBaseURL = cast.ToString(GetOrReturnDefaultValue("FILE_BASE_URL", "http://localhost:8080/v1/upload/"))
	c.Secret.UzumClientId = cast.ToString(GetOrReturnDefaultValue("UZUM_CLIENT_ID", "uzum_client_id"))
	c.Secret.UzumClientSecret = cast.ToString(GetOrReturnDefaultValue("UZUM_CLIENT_SECRET", "uzum_client_secret"))
	c.Secret.OAuthTokenExpiry = cast.ToInt(GetOrReturnDefaultValue("OAUTH_TOKEN_EXPIRY", 3600))

	c.Integration.ClickApiUrl = cast.ToString(GetOrReturnDefaultValue("CLICK_API_URL", "http://localhost:8080"))
	c.Integration.PaymeApiUrl = cast.ToString(GetOrReturnDefaultValue("PAYME_API_URL", "http://localhost:8080"))
	c.Integration.UzumApiUrl = cast.ToString(GetOrReturnDefaultValue("UZUM_API_URL", "http://localhost:8080"))
	c.Integration.AlifApiUrl = cast.ToString(GetOrReturnDefaultValue("ALIF_API_URL", "https://api-dev.alifpay.uz/v2"))
	c.Integration.AlifToken = cast.ToString(GetOrReturnDefaultValue("ALIF_TOKEN", "6fb5619d-8c30-4e85-a1e3-3f4d142498a0"))
	c.Integration.OnecApiUrl = cast.ToString(GetOrReturnDefaultValue("ONEC_API_URL", "http://localhost:8080"))
	c.Integration.OnecApiUsername = cast.ToString(GetOrReturnDefaultValue("ONEC_API_USERNAME", "pharma"))
	c.Integration.OnecApiPassword = cast.ToString(GetOrReturnDefaultValue("ONEC_API_PASSWORD", "password"))
	c.Integration.TasnifApiUrl = cast.ToString(GetOrReturnDefaultValue("TASNIF_API_URL", "http://localhost:8080"))
	c.Integration.NoorApiUrl = cast.ToString(GetOrReturnDefaultValue("NOOR_API_URL", "http://localhost:80"))
	c.Integration.NoorApiToken = cast.ToString(GetOrReturnDefaultValue("NOOR_API_TOKEN", "token"))
	c.Integration.DmedApiUrl = cast.ToString(GetOrReturnDefaultValue("DMED_API_URL", "http://localhost:80"))
	c.Integration.DmedApiToken = cast.ToString(GetOrReturnDefaultValue("DMED_API_TOKEN", "token"))
	c.Integration.OsonAptekaApiUrl = cast.ToString(GetOrReturnDefaultValue("OSON_APTEKA_API_URL", "https://remains.osonapteka.uz/api/set-app-remains"))
	return c
}

func GetOrReturnDefaultValue(key string, defaultValue interface{}) interface{} {
	val, exists := os.LookupEnv(key)
	if exists {
		return val
	}

	return defaultValue
}
