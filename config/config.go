package config

type Config struct {
	FrontendDomain string `env:"FRONTEND_DOMAIN" validate:"required"`
	FrontendURL    string `env:"FRONTEND_URL" validate:"required"`
	RepositoryPath string `env:"REPOSITORY_PATH" validate:"required"`
}
