package yubikey

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

// content holds our static web server content.
//
//go:embed templates
//go:embed static
var content embed.FS

func (s *Server) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", &WebData{Version: Version()})
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
