package server

import (
	// Stdlib
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"time"

	// Internal
	"github.com/salsaflow/salsaflow-server/server/common"

	// Vendor
	"github.com/codegangsta/negroni"
	noauth2 "github.com/goincremental/negroni-oauth2"
	sessions "github.com/goincremental/negroni-sessions"
	"github.com/goincremental/negroni-sessions/cookiestore"
	"github.com/gorilla/mux"
	"github.com/unrolled/secure"
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
	store          DataStore
	oauth2Config   *noauth2.Config
	addr           string
	cookieSecret   string
	rootDir        string
	timeout        time.Duration
}

type OptionFunc func(srv *Server)

func New(store DataStore, config *noauth2.Config, options ...OptionFunc) *Server {
	srv := &Server{
		store:        store,
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
	topRouter := mux.NewRouter()
	if srv.pathPrefix != "" {
		topRouter.PathPrefix(srv.pathPrefix).Handler(router)
	} else {
		topRouter = router
	}

	// Root.
	router.Handle("/", srv.loginRequired(http.HandlerFunc(srv.handleRootPath)))

	// Login.
	router.HandleFunc("/login", srv.handleLogin)

	// User profile.
	router.Handle("/profile", srv.loginRequired(http.HandlerFunc(srv.handleProfile)))

	// Configurations.
	router.Handle("/configurations", srv.loginRequired(http.HandlerFunc(srv.handleConfigurations)))

	// Commits.
	router.Handle("/commits", srv.loginRequired(http.HandlerFunc(srv.handleCommits)))

	// API.
	router.PathPrefix("/api/").Handler(http.StripPrefix("/api", srv.api()))

	// Assets.
	assets := http.FileServer(http.Dir(filepath.Join(srv.rootDir, "assets")))
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", assets))

	// Negroni middleware.
	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())

	n.UseFunc(secure.New(secure.Options{
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		IsDevelopment:   !srv.productionMode,
	}).HandlerFuncWithNext)

	n.Use(sessions.Sessions("SalsaFlowSession", cookiestore.New([]byte(srv.cookieSecret))))
	n.Use(noauth2.Google(srv.oauth2Config))
	n.UseHandler(topRouter)

	// Start the server using graceful.
	graceful.Run(srv.addr, srv.timeout, n)
}

func (srv *Server) handleRootPath(rw http.ResponseWriter, r *http.Request) {
	user, err := getPageRequester(r, srv.oauth2Config, srv.store)
	if err != nil {
		httpError(rw, r, err)
		return
	}

	if user != nil {
		http.Redirect(rw, r, srv.relativePath("/configurations"), http.StatusFound)
	} else {
		http.Redirect(rw, r, srv.relativePath("/login"), http.StatusFound)
	}
}

func (srv *Server) handleLogin(rw http.ResponseWriter, r *http.Request) {
	// Make sure the user is really not authenticated.
	user, err := getPageRequester(r, srv.oauth2Config, srv.store)
	if err != nil {
		httpError(rw, r, err)
		return
	}
	if user != nil {
		http.Redirect(rw, r, srv.relativePath("/"), http.StatusFound)
		return
	}

	// Get the next URL.
	next := r.FormValue("next")
	if next == "" {
		next = srv.relativePath("/")
	}

	// Read the template.
	t, err := srv.loadTemplates("login.html", "page_header.html", "page_footer.html")
	if err != nil {
		httpError(rw, r, err)
		return
	}

	// Render the template and write it into the response.
	var content bytes.Buffer
	ctx := struct {
		PathPrefix string
		User       *common.User
		Title      string
		LoginURL   string
	}{
		srv.pathPrefix,
		nil,
		"Login",
		srv.relativePath(fmt.Sprintf("/auth/google/login?next=%v", url.QueryEscape(next))),
	}
	if err := t.Execute(&content, ctx); err != nil {
		httpError(rw, r, err)
		return
	}
	io.Copy(rw, &content)
}

func (srv *Server) handleProfile(rw http.ResponseWriter, r *http.Request) {
	// Get the user record.
	user, err := getPageRequester(r, srv.oauth2Config, srv.store)
	if err != nil {
		httpError(rw, r, err)
		return
	}

	// Read the template.
	t, err := srv.loadTemplates("profile.html", "page_header.html", "page_footer.html")
	if err != nil {
		httpError(rw, r, err)
		return
	}

	// Render the template and write it into the response.
	var content bytes.Buffer
	ctx := struct {
		PathPrefix string
		Title      string
		User       *common.User
		LogoutURL  string
	}{
		srv.pathPrefix,
		"Profile",
		user,
		srv.relativePath("/auth/google/logout?next=") + url.QueryEscape(srv.relativePath("/login")),
	}
	if err := t.Execute(&content, ctx); err != nil {
		httpError(rw, r, err)
		return
	}
	io.Copy(rw, &content)
}

func (srv *Server) handleConfigurations(rw http.ResponseWriter, r *http.Request) {
	user, err := getPageRequester(r, srv.oauth2Config, srv.store)
	if err != nil {
		httpError(rw, r, err)
		return
	}

	// Read the template.
	t, err := srv.loadTemplates("configurations.html", "page_header.html", "page_footer.html")
	if err != nil {
		httpError(rw, r, err)
		return
	}

	// Render the template and write it into the response.
	var content bytes.Buffer
	ctx := struct {
		PathPrefix string
		Title      string
		User       *common.User
		LogoutURL  string
	}{
		srv.pathPrefix,
		"Configurations",
		user,
		srv.relativePath("/auth/google/logout?next=") + url.QueryEscape(srv.relativePath("/login")),
	}
	if err := t.Execute(&content, ctx); err != nil {
		httpError(rw, r, err)
		return
	}
	io.Copy(rw, &content)
}

func (srv *Server) handleCommits(rw http.ResponseWriter, r *http.Request) {
	user, err := getPageRequester(r, srv.oauth2Config, srv.store)
	if err != nil {
		httpError(rw, r, err)
		return
	}

	// Read the template.
	t, err := srv.loadTemplates("commits.html", "page_header.html", "page_footer.html")
	if err != nil {
		httpError(rw, r, err)
		return
	}

	// Render the template and write it into the response.
	var content bytes.Buffer
	ctx := struct {
		PathPrefix string
		Title      string
		User       *common.User
		LogoutURL  string
	}{
		srv.pathPrefix,
		"Commits",
		user,
		srv.relativePath("/auth/google/logout?next=") + url.QueryEscape(srv.relativePath("/login")),
	}
	if err := t.Execute(&content, ctx); err != nil {
		httpError(rw, r, err)
		return
	}
	io.Copy(rw, &content)
}

func (srv *Server) api() http.Handler {
	api := &API{srv.store}

	router := mux.NewRouter()
	top := mux.NewRouter()
	top.PathPrefix("/v1").Handler(http.StripPrefix("/v1", router))

	router.Path("/v1/me").Methods("GET").HandlerFunc(api.GetMe)
	router.Path("/v1/users/{userId}/generateToken").Methods("GET").HandlerFunc(api.GetGenerateToken)

	return top
}

func (srv *Server) loginRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var (
			session = sessions.GetSession(r)
			token   = noauth2.GetToken(r)
		)
		if token == nil || !token.Valid() {
			deleteProfile(session)
			noauth2.SetToken(r, nil)
			next := url.QueryEscape(r.URL.RequestURI())
			http.Redirect(rw, r, fmt.Sprintf("%v?next=%v", srv.relativePath("/login"), next), http.StatusFound)
		} else {
			next.ServeHTTP(rw, r)
		}
	})
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
