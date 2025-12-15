package httpserver

import (
	"context"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Server struct {
	httpServer *http.Server
	log        *zap.Logger
}

type Config struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func New(handler http.Handler, cfg Config, log *zap.Logger) *Server {
	return &Server{
		log: log,
		httpServer: &http.Server{
			Addr:         cfg.Addr,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
	}
}

func (s *Server) ListenAndServe() error {
	s.log.Info("http server binding",
		zap.String("addr", s.httpServer.Addr),
	)

	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		s.log.Error("http server bind failed", zap.Error(err))
		return err
	}

	s.log.Info("http server listening",
		zap.String("addr", s.httpServer.Addr),
	)

	err = s.httpServer.Serve(ln)
	if err != nil && err != http.ErrServerClosed {
		s.log.Error("http server serve failed", zap.Error(err))
		return err
	}

	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http server shutdown started")

	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.log.Error("http server shutdown failed", zap.Error(err))
		return err
	}

	s.log.Info("http server shutdown completed")
	return nil
}
