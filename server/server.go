package server

import (
	// Stdlib
	"fmt"
	"log"
	"net/http"
	"time"

	// Vendor
	"github.com/codegangsta/negroni"
	oauth2 "github.com/goincremental/negroni-oauth2"
	sessions "github.com/goincremental/negroni-sessions"
	"github.com/goincremental/negroni-sessions/cookiestore"
	"github.com/gorilla/mux"
	"google.golang.org/api/plus/v1"
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
		// Redirect to /login in case the user is not logged in.
		token := oauth2.GetToken(r)
		if token == nil || !token.Valid() {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		// Print something retarded.
		writeUserEmail(token)
	})

	// Restricted section.
	secureRouter := mux.NewRouter()

	secure := negroni.New()
	secure.Use(oauth2.LoginRequired())
	secure.UseHandler(secureRouter)

	router.Handle("/restricted", secure)

	// Negroni.
	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.Use(sessions.Sessions("SalsaFlowSession", cookiestore.New([]byte("SalsaFlow123"))))
	n.Use(oauth2.Google(srv.oauth2Config))
	n.UseHandler(router)

	// Start the server using graceful.
	graceful.Run(srv.addr, srv.timeout, n)
}

func writeUserEmail(w http.ResponseWriter, token oauth2.Token) {
	httpClient := NewOAuth2HttpClient(token)
	srv := plus.New(httpClient)

	people, err := plus.NewPeopleService(srv)
	if err != nil {
		nuke(w, err)
		return
	}

	me, err := people.Get("me").Do()
	if err != nil {
		nuke(w, err)
		return
	}

	fmt.Printf("%+v\n", me)
	fmt.Fprintf(w, "%+v\n", me)
}

func nuke(w http.ResponseWriter, err error) {
	log.Log(err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
