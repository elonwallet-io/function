package main

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/config"
	"github.com/Leantar/elonwallet-function/server"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if err := run(); err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to start")
	}
}

func run() error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill, syscall.SIGTERM)

	var cfg config.Config
	err := config.FromEnv(&cfg)
	if err != nil {
		return err
	}

	err = validator.New().Struct(cfg)
	if err != nil {
		return fmt.Errorf("validation of config failed: %w", err)
	}

	s := server.New(cfg)
	go func() {
		err := s.Run()
		if err != nil {
			panic(err)
		}
	}()

	<-stop

	return s.Stop()
}
