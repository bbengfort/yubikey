package yubikey

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bbengfort/yubikey/config"
	"github.com/bbengfort/yubikey/logger"
	"github.com/bbengfort/yubikey/session"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// Initializes zerolog with our default logging requirements
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = logger.GCPFieldKeyTime
	zerolog.MessageFieldName = logger.GCPFieldKeyMsg

	// Add the severity hook for GCP logging
	var gcpHook logger.SeverityHook
	log.Logger = zerolog.New(os.Stdout).Hook(gcpHook).With().Timestamp().Logger()
}

func New(conf config.Config) (s *Server, err error) {
	// Load the default configuration from the environment if config is empty.
	if conf.IsZero() {
		if conf, err = config.New(); err != nil {
			return nil, err
		}
	}

	// Setup our logging config first thing
	zerolog.SetGlobalLevel(conf.GetLogLevel())
	if conf.ConsoleLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Create the service and register it with the server.
	s = &Server{
		conf:    conf,
		errc:    make(chan error, 1),
		healthy: false,
		ready:   false,
		users:   NewUsers(),
	}

	// Create the webauthn instance
	if s.authn, err = webauthn.New(s.conf.WebAuthn.Config()); err != nil {
		return nil, err
	}

	// Create the session store
	if s.sessions, err = session.New(); err != nil {
		return nil, err
	}

	// Create the Gin router and setup its routes
	gin.SetMode(conf.Mode)
	s.router = gin.New()
	s.router.RedirectTrailingSlash = true
	s.router.RedirectFixedPath = false
	s.router.HandleMethodNotAllowed = true
	s.router.ForwardedByClientIP = true
	s.router.UseRawPath = false
	s.router.UnescapePathValues = true
	if err = s.setupRoutes(); err != nil {
		return nil, err
	}

	// Create the http server
	s.srv = &http.Server{
		Addr:         s.conf.BindAddr,
		Handler:      s.router,
		ErrorLog:     nil,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  120 * time.Second,
		TLSConfig:    s.conf.TLS.Config(),
	}
	return s, nil
}

type Server struct {
	sync.RWMutex
	conf     config.Config      // configuration of the API server
	authn    *webauthn.WebAuthn // the passwordless authentication module
	srv      *http.Server       // handle to a custom http server with specified API defaults
	users    *Users             // the users "database" for testing registration
	sessions *session.Store     // the sessions "database" for testing registration and login
	router   *gin.Engine        // the http handler and associated middlware
	healthy  bool               // application state of the server for health checks
	ready    bool               // application state of the server for ready checks
	started  time.Time          // the timestamp when the server was started
	url      *url.URL           // the url of the service when it's running
	errc     chan error         // synchronize shutdown gracefully
}

func (s *Server) Serve() (err error) {
	// Handle OS Signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		s.errc <- s.Shutdown()
	}()

	// Create a socket to listen on and infer the final URL.
	// NOTE: if the bindaddr is 127.0.0.1:0 for testing, a random port will be assigned,
	// manually creating the listener will allow us to determine which port.
	// When we start listening all incoming requests will be buffered until the server
	// actually starts up in its own go routine below.
	var sock net.Listener
	if sock, err = net.Listen("tcp", s.srv.Addr); err != nil {
		return fmt.Errorf("could not listen on bind addr %s: %s", s.srv.Addr, err)
	}

	s.SetStatus(true, true)
	s.started = time.Now()
	s.setURL(sock.Addr())

	if s.conf.Maintenance {
		log.Warn().Msg("starting server in maintenance mode")
	}

	// Listen for HTTP requests and handle them.
	go func() {
		// Make sure we don't use the external err to avoid data races.
		if serr := s.serve(sock); !errors.Is(serr, http.ErrServerClosed) {
			s.errc <- serr
		}

		// If there is no error, return nil so this function exits if Shutdown is
		// called manually (e.g. not from an OS signal).
		s.errc <- nil
	}()

	log.Info().Str("url", s.URL()).Msg("yubikey authn server started")
	return <-s.errc
}

// ServeTLS if a tls configuration is provided, otherwise Serve
func (s *Server) serve(sock net.Listener) error {
	if s.srv.TLSConfig != nil {
		return s.srv.ServeTLS(sock, "", "")
	}
	return s.srv.Serve(sock)
}

func (s *Server) Shutdown() error {
	log.Info().Msg("gracefully shutting down yubikey authn server")
	s.SetStatus(false, false)

	errs := make([]error, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	s.srv.SetKeepAlivesEnabled(false)
	if err := s.srv.Shutdown(ctx); err != nil {
		errs = append(errs, err)
	}

	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return fmt.Errorf("%d errors occurred during shutdown", len(errs))
	}
}

// SetHealth sets the health status on the API server, putting it into unavailable mode
// if health is false, and removing maintenance mode if health is true. Here primarily
// for testing purposes since it is unlikely an outside caller can access this.
func (s *Server) SetStatus(health, ready bool) {
	s.Lock()
	s.healthy = health
	s.ready = ready
	s.Unlock()
	log.Debug().Bool("health", health).Bool("ready", ready).Msg("server status set")
}

// URL returns the URL of the server determined by the socket addr.
func (s *Server) URL() string {
	s.RLock()
	defer s.RUnlock()
	return s.url.String()
}

// Set the URL from the TCPAddr when the server is started. Should be set by Serve().
func (s *Server) setURL(addr net.Addr) {
	s.Lock()
	defer s.Unlock()
	s.url = &url.URL{
		Scheme: "http",
		Host:   addr.String(),
	}

	if s.srv.TLSConfig != nil {
		s.url.Scheme = "https"
	}

	if tcp, ok := addr.(*net.TCPAddr); ok && tcp.IP.IsUnspecified() {
		s.url.Host = fmt.Sprintf("127.0.0.1:%d", tcp.Port)
	}
}
