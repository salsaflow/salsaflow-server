package server

import (
	// Stdlib
	"time"

	// Vendor
	"github.com/codegangsta/negroni"
	oauth2 "github.com/goincremental/negroni-oauth2"
	sessions "github.com/goincremental/negroni-sessions"
	"github.com/goincremental/negroni-sessions/cookiestore"
	"github.com/gorilla/mux"
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
	// Top-level router.
	router := mux.NewRouter()

	// Restricted section.
	secureRouter := mux.NewRouter()

	secure := negroni.New()
	secure.Use(oauth2.LoginRequired())
	secure.UseHandler(secureRouter)

	router.Handle("/restricted", secure)

	// Negroni.
	n := negroni.Classic()
	n.Use(sessions.Sessions("SalsaFlowSession", cookiestore.New([]byte("SalsaFlow123"))))
	n.Use(oauth2.Google(&oauth2.Config{
		ClientID:     "",
		ClientSecret: "",
		RedirectURL:  "",
		Scopes:       []string{},
	}))
	n.UseHandler(router)

	// Start the server using graceful.
	graceful.Run(srv.addr, srv.timeout, n)
}
