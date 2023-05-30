package server

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/Leantar/elonwallet-function/server/handlers"
	customMiddleware "github.com/Leantar/elonwallet-function/server/middleware"
)

func (s *Server) registerRoutes() error {
	api, err := handlers.NewApi(s.cfg, s.repo, s.key)
	if err != nil {
		return fmt.Errorf("failed to create new api: %w", err)
	}

	s.echo.GET("/register/initialize", api.HandleRegisterInitialize())
	s.echo.POST("/register/finalize", api.HandleRegisterFinalize())

	s.echo.GET("/login/initialize", api.HandleLoginInitialize())
	s.echo.POST("/login/finalize", api.HandleLoginFinalize())

	s.echo.GET("/logout", api.HandleLogout(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))

	s.echo.GET("/fees", api.HandleEstimateFees(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))

	s.echo.GET("/credentials/initialize", api.HandleCreateCredentialInitialize(), customMiddleware.CheckStrictAuthentication(s.repo, s.key.PublicKey, common.ScopeUser, common.ScopeCreateCredential))
	s.echo.POST("/credentials/finalize", api.HandleCreateCredentialFinalize(), customMiddleware.CheckStrictAuthentication(s.repo, s.key.PublicKey, common.ScopeUser, common.ScopeCreateCredential))
	s.echo.DELETE("/credentials/:name", api.HandleRemoveCredential(), customMiddleware.CheckStrictAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.GET("/credentials", api.HandleGetCredentials(), customMiddleware.CheckStrictAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))

	s.echo.POST("/wallets", api.HandleCreateWallet(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.GET("/wallets", api.HandleGetWallets(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))

	s.echo.GET("/networks", api.HandleGetNetworks(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))

	s.echo.GET("/jwt-verification-key", api.HandleGetJWTVerificationKey())

	s.echo.GET("/otp", api.HandleGetOTP(), customMiddleware.CheckStrictAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.POST("/otp/login", api.HandleLoginWithOTP())

	s.echo.POST("/message/sign", api.HandleSignPersonal(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.POST("/typed-data/sign", api.HandleSignTypedData(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))

	s.echo.POST("/transaction/sign/initialize", api.HandleSignTransactionInitialize(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.POST("/transaction/sign/finalize", api.HandleSignTransactionFinalize(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.POST("/transaction/send/initialize", api.HandleSendTransactionInitialize(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.POST("/transaction/send/finalize", api.HandleSendTransactionFinalize(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))

	s.echo.POST("/emergency-access/contacts", api.HandleCreateEmergencyContact(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.GET("/emergency-access/contacts", api.HandleGetEmergencyContacts(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.DELETE("/emergency-access/contacts/:email", api.HandleRemoveEmergencyContact(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.POST("/emergency-access/contacts/grant-response", api.HandleEmergencyAccessGrantResponse(), customMiddleware.CheckEnclaveAuthentication(s.repo, s.cfg))
	s.echo.GET("/emergency-access/contacts/request-access", api.HandleEmergencyContactAccessRequest(), customMiddleware.CheckEnclaveAuthentication(s.repo, s.cfg))
	s.echo.GET("/emergency-access/contacts/request-takeover", api.HandleEmergencyContactTakeoverRequest(), customMiddleware.CheckEnclaveAuthentication(s.repo, s.cfg))
	s.echo.POST("emergency-access/contacts/:email/deny-access", api.HandleDenyEmergencyContactAccessRequest(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))

	s.echo.POST("/emergency-access/grants", api.HandleEmergencyAccessGrantInvitation(), customMiddleware.CheckEnclaveAuthentication(s.repo, s.cfg))
	s.echo.GET("/emergency-access/grants", api.HandleGetEmergencyAccessGrants(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.POST("/emergency-access/grants/respond-invitation", api.HandleRespondEmergencyAccessGrantInvitation(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.POST("/emergency-access/grants/request-access", api.HandleRequestEmergencyAccess(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.POST("/emergency-access/grants/request-takeover", api.HandleRequestEmergencyAccessTakeover(), customMiddleware.CheckAuthentication(s.repo, s.key.PublicKey, common.ScopeUser))
	s.echo.DELETE("/emergency-access/grants", api.HandleEmergencyAccessGrantRemoval(), customMiddleware.CheckEnclaveAuthentication(s.repo, s.cfg))
	s.echo.POST("/emergency-access/grants/deny-access-request", api.HandleEmergencyAccessRequestDenial(), customMiddleware.CheckEnclaveAuthentication(s.repo, s.cfg))

	return nil
}
