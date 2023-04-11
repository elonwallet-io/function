package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func RequestLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
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
	})
}
