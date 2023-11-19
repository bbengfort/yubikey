package yubikey

import (
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/bbengfort/yubikey/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Setup the server's middleware and routes.
func (s *Server) setupRoutes() (err error) {
	// Setup HTML template renderer
	var html *template.Template
	if html, err = template.ParseFS(content, "templates/*.html"); err != nil {
		return err
	}
	s.router.SetHTMLTemplate(html)

	// Setup static content server
	var static fs.FS
	if static, err = fs.Sub(content, "static"); err != nil {
		return err
	}
	s.router.StaticFS("/static", http.FS(static))

	// Setup CORS configuration
	corsConf := cors.Config{
		AllowMethods: []string{"GET", "HEAD"},
		AllowHeaders: []string{"Origin", "Content-Length", "Content-Type"},
		AllowOrigins: s.conf.AllowOrigins,
		MaxAge:       12 * time.Hour,
	}

	// Application Middleware
	// NOTE: ordering is important to how middleware is handled
	middlewares := []gin.HandlerFunc{
		// Logging should be on the outside so we can record the correct latency of requests
		// NOTE: logging panics will not recover
		logger.GinLogger("yubikey", Version()),

		// Panic recovery middleware
		gin.Recovery(),

		// CORS configuration allows the front-end to make cross-origin requests
		cors.New(corsConf),

		// Mainenance mode handling
		s.Available(),
	}

	// Add the middleware to the router
	for _, middleware := range middlewares {
		if middleware != nil {
			s.router.Use(middleware)
		}
	}

	// Kubernetes liveness probes
	s.router.GET("/healthz", s.Healthz)
	s.router.GET("/livez", s.Healthz)
	s.router.GET("/readyz", s.Readyz)

	// NotFound and NotAllowed routes
	s.router.NoRoute(s.NotFound)
	s.router.NoMethod(s.NotAllowed)

	// Add the v1 API routes (currently the only version)
	v1 := s.router.Group("/v1")
	{
		// Heartbeat route
		v1.GET("/status", s.Status)
	}

	return nil
}
