package server

import (
	// Stdlib
	"bytes"
	"html/template"
	"io"
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
	ds             DataStore
	oauth2Config   *noauth2.Config
	addr           string
	cookieSecret   string
	rootDir        string
	timeout        time.Duration
}

type OptionFunc func(srv *Server)

func New(ds DataStore, config *noauth2.Config, options ...OptionFunc) *Server {
	srv := &Server{
		ds:           ds,
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
	router := mux.NewRouter().PathPrefix(srv.pathPrefix)

	// Root.
	router.HandleFunc("/", srv.loginRequired(srv.handleRootPath))

	// Login.
	router.HandleFunc("/login/", srv.handleLogin)

	// Commits.
	// router.HandleFunc("/commits", srv.loginRequired(srv.handleCommits))

	// API.
	router.PathPrefix("/api/").Handler(srv.loginOrTokenRequired(srv.api()))

	// Assets.
	assets := http.FileServer(http.Dir(filepath.Join(srv.rootDir, "assets")))
	router.PathPrefix("/assets/").Handler(srv.loginRequired(http.StripPrefix("/assets/"), assets))

	// Negroni middleware.
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
	profile, err := srv.getProfile(r)
	if err != nil {
		httpError(rw, r, err)
		return
	}

	// Read the template.
	t, err := srv.loadTemplates("home.html", "page_header.html", "page_footer.html")
	if err != nil {
		httpError(w, r, err)
		return
	}

	// Render the template and write it into the response.
	var content bytes.Buffer
	ctx := struct {
		PathPrefix string
		Title      string
		UserName   string
		UserEmail  string
		LogoutURL  string
	}{
		srv.pathPrefix,
		"Home",
		profile.Name,
		profile.Email,
		srv.relativePath("/auth/google/logout?next=") + url.QueryEscape("/"),
	}
	if err := t.Execute(&content, ctx); err != nil {
		httpError(w, r, err)
		return
	}
	io.Copy(w, &content)
}

func (srv *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Read the template.
	t, err := srv.loadTemplates("login.html", "page_header.html", "page_footer.html")
	if err != nil {
		httpError(w, r, err)
		return
	}

	// Render the template and write it into the response.
	var content bytes.Buffer
	ctx := struct {
		PathPrefix string
		UserName   string
		Title      string
		LoginURL   string
	}{
		srv.pathPrefix,
		"",
		"Login",
		srv.relativePath("/auth/google/login"),
	}
	if err := t.Execute(&content, ctx); err != nil {
		httpError(w, r, err)
		return
	}
	io.Copy(w, &content)
}

func (srv *Server) api() http.Handler {
	// API routing.
	api := NewApi(srv.datastore)

	router := mux.NewRouter().PathPrefix("/v1")
	router.Path("/me").Methods("GET").HandleFunc(api.GetMe)
	router.Path("/users/{userId}/generateToken").Methods("GET").HandleFunc(api.GetGenerateToken)

	// Cover the whole API with token authentication.
	n := negroni.New(srv.authMiddleware())
	n.UseHandler(router)
	return n
}

func (srv *Server) getProfile(r *http.Request) (*userProfile, error) {
	// Get session for the given HTTP request.
	session := sessions.GetSession(r)

	// Get the user profile from the session.
	profile, err := unmarshalProfile(session)
	if err != nil {
		return nil, err
	}

	// In case there is no profile, fetch it and store it in the session.
	if profile == nil {
		var (
			cfg = (*oauth2.Config)(srv.oauth2Config)
			tok = (oauth2.Token)(token.Get())
		)
		profile, err = fetchProfile(cfg, &tok)
		if err != nil {
			return nil, err
		}
		if err := marshalProfile(s, profile); err != nil {
			return nil, err
		}
	}

	// Return the user profile.
	return profile, nil
}

func (srv *Server) loginRequired(next http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var (
			session = sessions.GetSession(r)
			token   = noauth2.GetToken(r)
		)
		if token == nil || !token.Valid() {
			deleteProfile(session)
			noauth2.SetToken(r, nil)
			http.Redirect(rw, r, srv.relativePath("/login"), http.StatusTemporaryRedirect)
		} else {
			next(rw, r)
		}
	}
}

func (srv *Server) loginOrTokenRequired(next http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// Try the session token.
		token := noauth2.GetToken(r)
		if token != nil && token.Valid() {
			next(rw, r)
			return
		}

		// Try the access token.
		accessToken := r.Headers().Get("X-SalsaFlow-Token")
		if accessToken != "" {
			user, err := srv.store.FindUserByToken()
			if err != nil {
				httpError(rw, r, err)
				return
			}
			if user != nil {
				next(rw, r)
				return
			}
		}

		// Otherwise, unauthorized.
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
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
