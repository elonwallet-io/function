package config

type Config struct {
	FrontendHost string `env:"FRONTEND_HOST" validate:"required"`
	FrontendURL  string `env:"FRONTEND_URL" validate:"required"`
}
