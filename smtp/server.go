package smtp

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"path/filepath"

	"github.com/mailhog/MailHog-MTA/config"
	"github.com/mailhog/backends/auth"
	"github.com/mailhog/backends/delivery"
	"github.com/mailhog/backends/resolver"
)

// Server represents an SMTP server instance
type Server struct {
	BindAddr  string
	Hostname  string
	PolicySet config.ServerPolicySet

	AuthBackend     auth.Service
	DeliveryBackend delivery.Service
	ResolverBackend resolver.Service

	tlsConfig *tls.Config

	Config *config.Config
	Server *config.Server
}

func (s *Server) getTLSConfig() *tls.Config {
	if s.tlsConfig != nil {
		return s.tlsConfig
	}
	certPath := filepath.Join(s.Config.RelPath(), s.Server.TLSConfig.CertFile)
	keyPath := filepath.Join(s.Config.RelPath(), s.Server.TLSConfig.KeyFile)
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Fatal(err)
	}
	s.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	return s.tlsConfig
}

// Listen starts listening on the configured bind address
func (s *Server) Listen() error {
	log.Printf("[SMTP] Binding to address: %s\n", s.BindAddr)
	ln, err := net.Listen("tcp", s.BindAddr)
	if err != nil {
		log.Fatalf("[SMTP] Error listening on socket: %s\n", err)
		return err
	}

	defer ln.Close()

	sem := make(chan int, s.PolicySet.MaximumConnections)

	for {
		sem <- 1

		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[SMTP] Error accepting connection: %s\n", err)
			continue
		}

		go func() {
			s.Accept(
				conn.(*net.TCPConn).RemoteAddr().String(),
				io.ReadWriteCloser(conn),
			)

			<-sem
		}()
	}
}
