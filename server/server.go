package server

import (
	// Stdlib
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"time"

	// Vendor
	"github.com/codegangsta/negroni"
	noauth2 "github.com/goincremental/negroni-oauth2"
	sessions "github.com/goincremental/negroni-sessions"
	"github.com/goincremental/negroni-sessions/cookiestore"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	"gopkg.in/tylerb/graceful.v1"
)

const (
	DefaultAddress = "localhost:3000"
	DefaultTimeout = 3 * time.Second
)

type Server struct {
	pathPrefix   string
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
	// Set global negroni-oauth2 paths. Not too cool, to have global config in the package.
	noauth2.PathLogin = srv.relativePath("/auth/google/login")
	noauth2.PathLogout = srv.relativePath("/auth/google/logout")
	noauth2.PathCallback = srv.relativePath("/auth/google/callback")
	noauth2.PathError = srv.relativePath("/auth/google/error")

	// Top-level router.
	router := mux.NewRouter()
	router.HandleFunc("/", srv.handleRootPath)
	router.HandleFunc("/login", srv.handleLogin)

	// Restricted section.
	secureRouter := mux.NewRouter()

	secure := negroni.New()
	secure.Use(noauth2.LoginRequired())
	secure.UseHandler(secureRouter)

	router.Handle("/", secure)

	// Negroni.
	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.Use(sessions.Sessions("SalsaFlowSession", cookiestore.New([]byte("SalsaFlow123"))))
	n.Use(noauth2.Google(srv.oauth2Config))
	n.UseHandler(router)

	// Start the server using graceful.
	graceful.Run(srv.addr, srv.timeout, n)
}

func (srv *Server) handleRootPath(w http.ResponseWriter, r *http.Request) {
	// Redirect to /login in case the user is not logged in.
	token := noauth2.GetToken(r)
	if token == nil || !token.Valid() {
		noauth2.SetToken(r, nil)
		http.Redirect(w, r, srv.relativePath("/login"), http.StatusTemporaryRedirect)
		return
	}

	// Get the user profile from the session.
	s := sessions.GetSession(r)
	profile, err := unmarshalProfile(s)
	if err != nil {
		httpError(w, r, err)
		return
	}
	if profile == nil {
		var (
			cfg = (*oauth2.Config)(srv.oauth2Config)
			tok = (*oauth2.Token)(&token)
		)
		profile, err = fetchProfile(cfg, tok)
		if err != nil {
			httpError(w, r, err)
			return
		}
		if err := marshalProfile(s, profile); err != nil {
			httpError(w, r, err)
			return
		}
	}

	// Print the profile.
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
	<head>
		<title>SalsaFlow</title>
	</head>
	<body>
		<table>
			<tr><td>%v</td></tr>
			<tr><td>%v</td></tr>
		</table>
	</body>
</html>`, profile.Name, profile.Email)
}

func (srv *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
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
}

func (srv *Server) relativePath(pth string) string {
	return path.Join(srv.pathPrefix, pth)
}

func httpError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("[ERROR] %v %v -> %v\n", r.Method, r.URL.Path, err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
