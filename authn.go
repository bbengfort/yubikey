package yubikey

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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
	fmt.Println(session)

	// Return the options as JSON
	c.JSON(http.StatusOK, opts)
}

func (s *Server) FinishRegistration(c *gin.Context) {}
