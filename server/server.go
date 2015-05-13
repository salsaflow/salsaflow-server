package server

import (
	// Stdlib
	"time"

	// Vendor
	"github.com/codegangsta/negroni"
	"gopkg.in/tylerb/graceful.v1"
)

const (
	DefaultAddress = ":3000"
	DefaultTimeout = 3 * time.Second
)

type Server struct {
	addr    string
	timeout time.Duration
}

type OptionFunc func(srv *Server)

func New(options ...OptionFunc) *Server {
	srv := &Server{
		addr:    DefaultAddress,
		timeout: DefaultTimeout,
	}

	for _, opt := range options {
		opt(srv)
	}

	return srv
}

func SetAddress(addr string) OptionFunc {
	return func(srv *Server) {
		srv.addr = addr
	}
}

func SetShutdownTimeout(timeout time.Duration) OptionFunc {
	return func(srv *Server) {
		srv.timeout = timeout
	}
}

func (srv *Server) Run() {
	n := negroni.Classic()
	graceful.Run(srv.addr, srv.timeout, n)
}
