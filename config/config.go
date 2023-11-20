package config

import (
	"crypto/tls"
	"fmt"

	"github.com/bbengfort/yubikey/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rotationalio/confire"
	"github.com/rs/zerolog"
)

type Config struct {
	Maintenance  bool                `default:"false"`
	BindAddr     string              `split_words:"true" default:":443"`
	Mode         string              `default:"release"`
	LogLevel     logger.LevelDecoder `split_words:"true" default:"info"`
	ConsoleLog   bool                `split_words:"true" default:"false"`
	AllowOrigins []string            `split_words:"true" default:"https://yubikey.local"`
	WebAuthn     WebAuthnConfig      `split_words:"true"`
	TLS          TLSConfig
	processed    bool // set when the config is properly processed from the environment
}

type TLSConfig struct {
	UseTLS   bool   `split_words:"true" default:"false"`
	CertFile string `split_words:"true" default:"tmp/server.crt"`
	KeyFile  string `split_words:"true" default:"tmp/server.key"`
}

type WebAuthnConfig struct {
	RPID        string   `default:"yubikey.local"`
	DisplayName string   `split_words:"true" default:"Yubikey Authn Debugger"`
	Origins     []string `default:"https://yubikey.local"`
}

func New() (conf Config, err error) {
	if err = confire.Process("yubikey", &conf); err != nil {
		return Config{}, err
	}

	if err = conf.Validate(); err != nil {
		return Config{}, err
	}

	conf.processed = true
	return conf, nil
}

// Returns true if the config has not been correctly processed from the environment.
func (c Config) IsZero() bool {
	return !c.processed
}

// Custom validations are added here, particularly validations that require one or more
// fields to be processed before the validation occurs.
// NOTE: ensure that all nested config validation methods are called here.
func (c Config) Validate() (err error) {
	if c.Mode != gin.ReleaseMode && c.Mode != gin.DebugMode && c.Mode != gin.TestMode {
		return fmt.Errorf("invalid configuration: %q is not a valid gin mode", c.Mode)
	}
	return nil
}

func (c Config) GetLogLevel() zerolog.Level {
	return zerolog.Level(c.LogLevel)
}

func (c TLSConfig) Config() *tls.Config {
	if c.UseTLS {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			panic(err)
		}

		return &tls.Config{Certificates: []tls.Certificate{cert}}
	}
	return nil
}

func (c WebAuthnConfig) Config() *webauthn.Config {
	return &webauthn.Config{
		RPID:          c.RPID,
		RPDisplayName: c.DisplayName,
		RPOrigins:     c.Origins,
	}
}
