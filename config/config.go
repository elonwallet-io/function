package config

type Config struct {
	Server ServerConfig
}

type ServerConfig struct {
	CorsAllowedUrl string `env:"CORS_ALLOWED_URL" validate:"required"`
}
