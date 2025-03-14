package socket

import (
	"crypto/tls"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type apiSettings struct {
}

func registerDefaultAPISettings(settings *apiSettings) error {
	return nil
}

type settings struct {
	// Server settings
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	MaxHeaderBytes    int
	ShutdownTimeout   time.Duration

	// TLS configuration
	TLSConfig   *tls.Config
	TLSCertFile string
	TLSKeyFile  string

	// Connection settings
	MaxConnections    int
	ConnectionTimeout time.Duration

	// WebSocket specific
	PingInterval     time.Duration
	PongWait         time.Duration
	WriteWait        time.Duration
	MessageSizeLimit int64

	// Router settings
	BasePath         string
	EnableCORS       bool
	CORSAllowOrigins []string
	CORSAllowMethods []string
	CORSAllowHeaders []string

	// Middleware
	EnableLogging     bool
	EnableCompression bool
	RateLimiter       *rate.Limiter
}

func (s *settings) apply(socket *Socket) {
	socket.server.ReadTimeout = s.ReadTimeout
	socket.server.WriteTimeout = s.WriteTimeout
	socket.server.IdleTimeout = s.IdleTimeout
	socket.server.ReadHeaderTimeout = s.ReadHeaderTimeout
	socket.server.MaxHeaderBytes = s.MaxHeaderBytes

	socket.server.TLSConfig = s.TLSConfig

	if s.EnableCORS {
		// s.applyCORS()
	}

	if s.EnableLogging {

	}

	if s.EnableCompression {

	}
}

func (s *settings) applyCORS(handler *http.HandlerFunc) {

}

func registerDefaultSettings(settings *settings) error {
	return nil
}
