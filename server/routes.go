package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/repository"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
)

func (s *Server) registerRoutes() error {
	d := repository.NewJsonFile(".")

	//TODO clean this up
	signingKey, err := d.GetSigningKey()
	if os.IsNotExist(err) {
		pk, sk, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			log.Fatal().Caller().Err(err).Msg("failed to generate jwt signing key")
		}
		signingKey = models.SigningKey{
			PrivateKey: sk,
			PublicKey:  pk,
		}
		err = d.SaveSigningKey(signingKey)
		if err != nil {
			log.Fatal().Caller().Err(err).Msg("failed to save jwt signing key")
		}
	} else if err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to get jwt signing key")
	}

	api, err := NewApi(d, signingKey)
	if err != nil {
		return fmt.Errorf("failed to create new api: %w", err)
	}

	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{s.cfg.Server.CorsAllowedUrl},
		AllowMethods:     []string{http.MethodHead, http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPut},
		AllowCredentials: true,
	}))
	s.echo.GET("/register/initialize", api.RegisterInitialize())
	s.echo.POST("/register/finalize", api.RegisterFinalize(), api.manageUser())

	s.echo.GET("/login/initialize", api.LoginInitialize(), api.manageUser())
	s.echo.POST("/login/finalize", api.LoginFinalize(), api.manageUser())

	s.echo.GET("/transaction/initialize", api.TransactionInitialize(), api.checkAuthentication(), api.manageUser())
	s.echo.POST("/transaction/finalize", api.TransactionFinalize(), api.checkAuthentication(), api.manageUser())
	s.echo.GET("/fees", api.EstimateFees())

	s.echo.GET("/credentials/initialize", api.CreateCredentialInitialize(), api.checkAuthentication(), api.manageUser())
	s.echo.POST("/credentials/finalize", api.CreateCredentialFinalize(), api.checkAuthentication(), api.manageUser())
	s.echo.DELETE("/credentials/:name", api.RemoveCredential(), api.checkAuthentication(), api.manageUser())
	s.echo.GET("/credentials", api.GetCredentials(), api.checkAuthentication(), api.manageUser())

	s.echo.POST("/wallets", api.CreateWallet(), api.checkAuthentication(), api.manageUser())
	s.echo.GET("/wallets", api.GetWallets(), api.checkAuthentication(), api.manageUser())

	s.echo.GET("/networks", api.GetNetworks(), api.checkAuthentication())

	s.echo.GET("/jwt-verification-key", api.HandleGetJWTVerificationKey())

	return nil
}
