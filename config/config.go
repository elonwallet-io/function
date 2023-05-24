package config

type Config struct {
	FrontendHost    string `env:"FRONTEND_HOST" validate:"required"`
	FrontendURL     string `env:"FRONTEND_URL" validate:"required"`
	BackendURL      string `env:"BACKEND_URL" validate:"required"`
	UseInsecureHTTP bool   `env:"USE_INSECURE_HTTP"`
}
