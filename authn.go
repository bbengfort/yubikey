package yubikey

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
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
		if errors.Is(err, ErrUserAlreadyExists) {
			if user, err = s.users.GetUser(form.Email); err != nil {
				log.Warn().Err(err).Msg("could not retrieve an existing user")
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Ensure the same authenticator cannot be registered twice
	registerOptions := func(credCreationOpts *protocol.PublicKeyCredentialCreationOptions) {
		credCreationOpts.CredentialExcludeList = user.CredentialExcludeList()
	}

	opts, session, err := s.authn.BeginRegistration(user, registerOptions)
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

	// Check to make sure the credential is not already assigned to a user.
	if s.users.CredentialExists(credential) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "credential already assigned"})
		return
	}

	// Add the credential to the user and return the response
	user.AddCredential(*credential)
	c.JSON(http.StatusOK, gin.H{"message": "registration successful"})
}

type LoginForm struct {
	Email string `json:"email"`
}

func (s *Server) BeginLogin(c *gin.Context) {
	form := &LoginForm{}
	if err := c.BindJSON(form); err != nil {
		log.Error().Err(err).Msg("could not bind login form")
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not bind login form"})
		return
	}

	if form.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required to login"})
		return
	}

	// Look up user
	user, err := s.users.GetUser(form.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	opts, session, err := s.authn.BeginLogin(user)
	if err != nil {
		log.Error().Err(err).Msg("could not begin webauthn login")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Session values must be stored
	if err := s.sessions.SaveWebauthnSession("authentication", session, c.Request, c.Writer); err != nil {
		log.Error().Err(err).Msg("could not save session data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, opts)
}

func (s *Server) FinishLogin(c *gin.Context) {
	var (
		user    *User
		session webauthn.SessionData
		err     error
	)

	// Load the session data
	if session, err = s.sessions.GetWebauthnSession("authentication", c.Request); err != nil {
		log.Warn().Err(err).Msg("could not get session data from request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load the user from the session user ID
	// TODO: do we have to sidechannel this information for security?
	if user, err = s.users.Lookup(session.UserID); err != nil {
		log.Warn().Err(err).Msg("could not lookup user from session data")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err = s.authn.FinishLogin(user, session, c.Request); err != nil {
		log.Warn().Err(err).Msg("could not finish registration")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "login successful"})
}
