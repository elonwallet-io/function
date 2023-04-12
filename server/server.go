package server

import (
	"context"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	customMiddleware "github.com/Leantar/elonwallet-function/server/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"time"

	"github.com/Leantar/elonwallet-function/config"
	"github.com/labstack/echo/v4"
)

type Server struct {
	echo *echo.Echo
	cfg  config.Config
	key  models.SigningKey
	repo common.Repository
}

func New(cfg config.Config, key models.SigningKey, repo common.Repository) *Server {
	e := echo.New()
	e.Server.ReadTimeout = 5 * time.Second
	e.Server.WriteTimeout = 10 * time.Second
	e.Server.IdleTimeout = 120 * time.Second

	cv := CustomValidator{
		validator: validator.New(),
	}
	e.Validator = &cv
	e.Binder = &BinderWithURLDecoding{&echo.DefaultBinder{}}

	e.Use(middleware.RequestID())
	e.Use(customMiddleware.RequestLogger())
	e.Use(customMiddleware.Cors(cfg.FrontendURL))

	return &Server{
		echo: e,
		cfg:  cfg,
		key:  key,
		repo: repo,
	}
}

func (s *Server) Run() (err error) {
	err = s.registerRoutes()
	if err != nil {
		return
	}

	err = s.echo.Start("0.0.0.0:8081")
	if err == http.ErrServerClosed {
		err = nil
	}

	return
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.echo.Shutdown(ctx)
}
