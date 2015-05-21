package server

import (
	// Stdlib
	"io"
	"net/http"
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
	DefaultAddress = "localhost:3000"
	DefaultTimeout = 3 * time.Second
)

type Server struct {
	oauth2Config *oauth2.Config
	addr         string
	timeout      time.Duration
}

type OptionFunc func(srv *Server)

func New(config *oauth2.Config, options ...OptionFunc) *Server {
	srv := &Server{
		oauth2Config: config,
		addr:         DefaultAddress,
		timeout:      DefaultTimeout,
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
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK")
	})

	// Restricted section.
	secureRouter := mux.NewRouter()

	secure := negroni.New()
	secure.Use(oauth2.LoginRequired())
	secure.UseHandler(secureRouter)

	router.Handle("/restricted", secure)

	// Negroni.
	n := negroni.Classic()
	n.Use(sessions.Sessions("SalsaFlowSession", cookiestore.New([]byte("SalsaFlow123"))))
	n.Use(oauth2.Google(srv.oauth2Config))
	n.UseHandler(router)

	// Start the server using graceful.
	graceful.Run(srv.addr, srv.timeout, n)
}
