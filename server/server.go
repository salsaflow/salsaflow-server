package server

import (
	// Stdlib
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"time"

	// Vendor
	"github.com/codegangsta/negroni"
	noauth2 "github.com/goincremental/negroni-oauth2"
	sessions "github.com/goincremental/negroni-sessions"
	"github.com/goincremental/negroni-sessions/cookiestore"
	"github.com/gorilla/mux"
	"github.com/unrolled/secure"
	"golang.org/x/oauth2"
	"gopkg.in/tylerb/graceful.v1"
)

const (
	DefaultAddress      = "localhost:3000"
	DefaultCookieSecret = "OneRingToRuleThemAll"
	DefaultTimeout      = 3 * time.Second
)

type Server struct {
	productionMode bool
	pathPrefix     string
	oauth2Config   *noauth2.Config
	addr           string
	cookieSecret   string
	rootDir        string
	timeout        time.Duration
}

type OptionFunc func(srv *Server)

func New(config *noauth2.Config, options ...OptionFunc) *Server {
	srv := &Server{
		oauth2Config: config,
		addr:         DefaultAddress,
		cookieSecret: DefaultCookieSecret,
		timeout:      DefaultTimeout,
	}

	for _, opt := range options {
		opt(srv)
	}

	return srv
}

func EnableProductionMode() OptionFunc {
	return func(srv *Server) {
		srv.productionMode = true
	}
}

func SetAddress(addr string) OptionFunc {
	return func(srv *Server) {
		srv.addr = addr
	}
}

func SetRootDirectory(rootDir string) OptionFunc {
	return func(srv *Server) {
		srv.rootDir = rootDir
	}
}

func SetCookieSecret(secret string) OptionFunc {
	return func(srv *Server) {
		srv.cookieSecret = secret
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

	// Negroni.
	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())

	n.UseFunc(secure.New(secure.Options{
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		IsDevelopment:   !srv.productionMode,
	}).HandlerFuncWithNext)

	n.Use(sessions.Sessions("SalsaFlowSession", cookiestore.New([]byte(srv.cookieSecret))))
	n.Use(noauth2.Google(srv.oauth2Config))
	n.UseHandler(router)

	// Start the server using graceful.
	graceful.Run(srv.addr, srv.timeout, n)
}

func (srv *Server) handleRootPath(w http.ResponseWriter, r *http.Request) {
	// Redirect to /login in case the user is not logged in.
	var (
		s     = sessions.GetSession(r)
		token = noauth2.GetToken(r)
	)
	if token == nil || !token.Valid() {
		deleteProfile(s)
		noauth2.SetToken(r, nil)
		http.Redirect(w, r, srv.relativePath("/login"), http.StatusTemporaryRedirect)
		return
	}

	// Get the user profile from the session.
	profile, err := unmarshalProfile(s)
	if err != nil {
		httpError(w, r, err)
		return
	}
	if profile == nil {
		var (
			cfg = (*oauth2.Config)(srv.oauth2Config)
			tok = (oauth2.Token)(token.Get())
		)
		profile, err = fetchProfile(cfg, &tok)
		if err != nil {
			httpError(w, r, err)
			return
		}
		if err := marshalProfile(s, profile); err != nil {
			httpError(w, r, err)
			return
		}
	}

	// Read the template.
	t, err := srv.loadTemplates("homePage.html", "page_header.html", "page_footer.html")
	if err != nil {
		httpError(w, r, err)
		return
	}

	// Render the template and write it into the response.
	ctx := struct {
		Title     string
		Name      string
		Email     string
		LogoutURL string
	}{
		"Home",
		profile.Name,
		profile.Email,
		srv.relativePath("/auth/google/logout?next=") + url.QueryEscape("/"),
	}
	t.Execute(w, ctx)
}

func (srv *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Read the template.
	t, err := srv.loadTemplates("login.html", "page_header.html", "page_footer.html")
	if err != nil {
		httpError(w, r, err)
		return
	}

	// Render the template and write it into the response.
	ctx := struct {
		LoginURL string
	}{
		srv.relativePath("/auth/google/login"),
	}
	t.Execute(w, ctx)
}

func (srv *Server) relativePath(pth string) string {
	return path.Join(srv.pathPrefix, pth)
}

func (srv *Server) loadTemplates(fileNames ...string) (*template.Template, error) {
	paths := make([]string, 0, len(fileNames))
	for _, fileName := range fileNames {
		paths = append(paths, filepath.Join(srv.rootDir, "templates", fileName))
	}
	return template.ParseFiles(paths...)
}

func httpError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("[ERROR] %v %v -> %v\n", r.Method, r.URL.Path, err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
