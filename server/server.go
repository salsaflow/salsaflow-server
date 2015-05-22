package server

import (
	// Stdlib
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	// Vendor
	"github.com/codegangsta/negroni"
	noauth2 "github.com/goincremental/negroni-oauth2"
	sessions "github.com/goincremental/negroni-sessions"
	"github.com/goincremental/negroni-sessions/cookiestore"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/plus/v1"
	"gopkg.in/tylerb/graceful.v1"
)

const (
	DefaultAddress = "localhost:3000"
	DefaultTimeout = 3 * time.Second
)

type Server struct {
	oauth2Config *noauth2.Config
	addr         string
	timeout      time.Duration
}

type OptionFunc func(srv *Server)

func New(config *noauth2.Config, options ...OptionFunc) *Server {
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
	// Set global negroni-oauth2 paths. Ugly as hell.
	noauth2.PathLogin = "/auth/google/login"
	noauth2.PathLogout = "/auth/google/logout"
	noauth2.PathCallback = "/auth/google/callback"
	noauth2.PathError = "/auth/google/error"

	// Top-level router.
	router := mux.NewRouter()

	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `
<!DOCTYPE html>
<html>
	<head>
		<title>Login</title>
	</head>
	<body>
		<a href="/auth/google/login">Google</a>
	</body>
</html>
		`)
	})

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Redirect to /login in case the user is not logged in.
		token := noauth2.GetToken(r)
		if token == nil || !token.Valid() {
			noauth2.SetToken(r, nil)
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		// Print something retarded.
		srv.writeUserEmail(w, token.Get())
	})

	// Restricted section.
	secureRouter := mux.NewRouter()

	secure := negroni.New()
	secure.Use(noauth2.LoginRequired())
	secure.UseHandler(secureRouter)

	router.Handle("/restricted", secure)

	// Negroni.
	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.Use(sessions.Sessions("SalsaFlowSession", cookiestore.New([]byte("SalsaFlow123"))))
	n.Use(noauth2.Google(srv.oauth2Config))
	n.UseHandler(router)

	// Start the server using graceful.
	graceful.Run(srv.addr, srv.timeout, n)
}

func (srv *Server) writeUserEmail(w http.ResponseWriter, token noauth2.Token) {
	config := (*oauth2.Config)(srv.oauth2Config)
	tok := (*oauth2.Token)(&token)
	httpClient := config.Client(context.Background(), tok)

	service, err := plus.New(httpClient)
	if err != nil {
		nuke(w, err)
		return
	}

	people := plus.NewPeopleService(service)
	me, err := people.Get("me").Do()
	if err != nil {
		nuke(w, err)
		return
	}

	fmt.Fprintln(w, `
<!DOCTYPE html>
<html>
	<head>
		<title>SalsaFlow</title>
	</head>
	<body>
	`)
	fmt.Fprintln(w, me.DisplayName)
	fmt.Fprintln(w, me.Domain)
	for _, email := range me.Emails {
		fmt.Fprintln(w, email.Value)
	}

	fmt.Fprintln(w, `<a href="/auth/google/logout?next=/login">Logout</a>`)
	fmt.Fprintln(w, "</body></html>")
}

func nuke(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
