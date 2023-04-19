package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/Leantar/elonwallet-function/config"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/repository"
	"github.com/Leantar/elonwallet-function/server"
	"github.com/Leantar/elonwallet-function/server/common"
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
	signal.Notify(stop, syscall.SIGTERM)

	var cfg config.Config
	err := config.FromEnv(&cfg)
	if err != nil {
		return err
	}

	err = validator.New().Struct(cfg)
	if err != nil {
		return fmt.Errorf("validation of config failed: %w", err)
	}

	repo := repository.NewJsonFile()
	signingKey, err := getSigningKey(repo)
	if err != nil {
		return err
	}

	s := server.New(cfg, signingKey, repo)
	go func() {
		err := s.Run()
		if err != nil {
			panic(err)
		}
	}()

	<-stop

	return s.Stop()
}

func getSigningKey(repo *repository.JsonFile) (models.SigningKey, error) {
	signingKey, err := repo.GetSigningKey()
	if errors.Is(err, common.ErrNotFound) {
		signingKey, err = generateSigningKey()
		if err != nil {
			return models.SigningKey{}, err
		}

		//Store the key for the next time we need to fetch it after restart
		err = repo.SaveSigningKey(signingKey)
		if err != nil {
			return models.SigningKey{}, fmt.Errorf("failed to save new signing key: %w", err)
		}
	} else if err != nil {
		return models.SigningKey{}, fmt.Errorf("failed to get signing key: %w", err)
	}

	return signingKey, nil
}

func generateSigningKey() (models.SigningKey, error) {
	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return models.SigningKey{}, fmt.Errorf("failed to generate signing key: %w", err)
	}

	return models.SigningKey{
		PrivateKey: sk,
		PublicKey:  pk,
	}, nil
}
