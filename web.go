package yubikey

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) Index(c *gin.Context) {
	data := &UserList{}
	data.Version = Version()

	for _, user := range s.users.users {
		data.Users = append(data.Users, struct {
			ID          string
			Name        string
			Email       string
			Credentials int
		}{
			ID:          user.ID.String(),
			Name:        user.Name,
			Email:       user.Email,
			Credentials: len(user.WebAuthnCredentials()),
		})
	}

	c.HTML(http.StatusOK, "index.html", data)
}

func (s *Server) Register(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", &WebData{Version: Version()})
}

func (s *Server) Login(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", &WebData{Version: Version()})
}

func (s *Server) NotFound(c *gin.Context) {
	c.String(http.StatusNotFound, http.StatusText(http.StatusNotFound))
}

func (s *Server) NotAllowed(c *gin.Context) {
	c.String(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}

type WebData struct {
	Version string
}

type UserList struct {
	WebData
	Users []struct {
		ID          string
		Name        string
		Email       string
		Credentials int
	}
}
