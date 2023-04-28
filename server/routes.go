package server

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/server/handlers"
	customMiddleware "github.com/Leantar/elonwallet-function/server/middleware"
)

func (s *Server) registerRoutes() error {
	api, err := handlers.NewApi(s.cfg, s.repo, s.key)
	if err != nil {
		return fmt.Errorf("failed to create new api: %w", err)
	}

	s.echo.GET("/register/initialize", api.RegisterInitialize())
	s.echo.POST("/register/finalize", api.RegisterFinalize())

	s.echo.GET("/login/initialize", api.LoginInitialize())
	s.echo.POST("/login/finalize", api.LoginFinalize())

	s.echo.GET("/logout", api.Logout(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey))

	s.echo.GET("/fees", api.EstimateFees(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey))

	s.echo.GET("/credentials/initialize", api.CreateCredentialInitialize(), customMiddleware.CheckStrictAuthentication(s.repo, s.key.PublicKey, "create-credential"))
	s.echo.POST("/credentials/finalize", api.CreateCredentialFinalize(), customMiddleware.CheckStrictAuthentication(s.repo, s.key.PublicKey, "create-credential"))
	s.echo.DELETE("/credentials/:name", api.RemoveCredential(), customMiddleware.CheckStrictAuthentication(s.repo, s.key.PublicKey))
	s.echo.GET("/credentials", api.GetCredentials(), customMiddleware.CheckStrictAuthentication(s.repo, s.key.PublicKey))

	s.echo.POST("/wallets", api.CreateWallet(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey))
	s.echo.GET("/wallets", api.GetWallets(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey))

	s.echo.GET("/networks", api.GetNetworks(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey))

	s.echo.GET("/jwt-verification-key", api.HandleGetJWTVerificationKey())

	s.echo.GET("/otp", api.GetOrCreateOTP(), customMiddleware.CheckStrictAuthentication(s.repo, s.key.PublicKey))
	s.echo.POST("/otp/login", api.LoginWithOTP())

	s.echo.POST("/message/sign", api.SignPersonal(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey))

	s.echo.GET("/transaction/sign/initialize", api.SignTransactionInitialize(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey))
	s.echo.POST("/transaction/sign/finalize", api.SignTransactionFinalize(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey))
	s.echo.GET("/transaction/send/initialize", api.SendTransactionInitialize(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey))
	s.echo.POST("/transaction/send/finalize", api.SendTransactionFinalize(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey))

	return nil
}
