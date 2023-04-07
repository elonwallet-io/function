package server

import (
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"

	"github.com/Leantar/elonwallet-function/config"
	"github.com/labstack/echo/v4"
)

type Server struct {
	echo *echo.Echo
	cfg  config.Config
}

func New(cfg config.Config) *Server {
	e := echo.New()
	e.Server.ReadTimeout = 5 * time.Second
	e.Server.WriteTimeout = 10 * time.Second
	e.Server.IdleTimeout = 120 * time.Second

	cv := CustomValidator{
		validator: validator.New(),
	}
	e.Validator = &cv
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency:   true,
		LogRemoteIP:  true,
		LogMethod:    true,
		LogURI:       true,
		LogRequestID: true,
		LogStatus:    true,
		LogError:     true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			var evt *zerolog.Event

			_, ok := v.Error.(*echo.HTTPError)
			if v.Error == nil || ok {
				evt = log.Debug()

			} else {
				evt = log.Warn()
			}
			evt.Str("request_id", v.RequestID).
				Dur("latency", v.Latency).
				Str("remote_ip", v.RemoteIP).
				Str("method", v.Method).
				Str("uri", v.URI).
				Int("status", v.Status).
				Err(v.Error).
				Msg("request")

			return nil
		},
	}))
	return &Server{
		echo: e,
		cfg:  cfg,
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
