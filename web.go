package yubikey

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", &WebData{Version: Version()})
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
