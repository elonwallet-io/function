package server

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/datastore"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
)

func (s *Server) registerRoutes() error {
	d := datastore.NewJsonFile(".")

	api, err := NewApi(d)
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

	s.echo.GET("/jwt-verification-key", api.HandleGetJWTVerificationKey(), api.manageUser())

	return nil
}
