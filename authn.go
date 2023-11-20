package yubikey

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rs/zerolog/log"
)

type RegistrationForm struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (s *Server) BeginRegistration(c *gin.Context) {

	form := &RegistrationForm{}
	if err := c.BindJSON(form); err != nil {
		log.Error().Err(err).Msg("could not bind registration form")
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not bind registration form"})
		return
	}

	if form.Email == "" || form.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing either name or email address"})
		return
	}

	// Find or create a new user
	user, err := s.users.NewUser(form.Name, form.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	opts, session, err := s.authn.BeginRegistration(user)
	if err != nil {
		log.Error().Err(err).Msg("could not begin webauthn registration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not begin webauthn registration"})
		return
	}

	// Session values must be stored
	if err := s.sessions.SaveWebauthnSession("registration", session, c.Request, c.Writer); err != nil {
		log.Error().Err(err).Msg("could not save session data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not begin webauthn registration"})
		return
	}

	// Return the options as JSON
	c.JSON(http.StatusOK, opts)
}

func (s *Server) FinishRegistration(c *gin.Context) {
	var (
		user    *User
		session webauthn.SessionData
		err     error
	)

	// Load the session data
	if session, err = s.sessions.GetWebauthnSession("registration", c.Request); err != nil {
		log.Warn().Err(err).Msg("could not get session data from request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load the user from the session user ID
	// TODO: do we have to sidechannel this information for security?
	// e.g. the example uses the username in a param rather than from the session
	if user, err = s.users.Lookup(session.UserID); err != nil {
		log.Warn().Err(err).Msg("could not lookup user from session data")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var credential *webauthn.Credential
	if credential, err = s.authn.FinishRegistration(user, session, c.Request); err != nil {
		log.Warn().Err(err).Msg("could not finish registration")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Add the credential to the user and return the response
	user.AddCredential(*credential)
	c.JSON(http.StatusOK, gin.H{"message": "registration successful"})
}

func (s *Server) BeginLogin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "coming soon"})
}

func (s *Server) FinishLogin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "coming soon"})
}
