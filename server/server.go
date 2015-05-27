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

	// Internal
	"github.com/salsaflow/salsaflow-server/server/common"

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

	// Commits.
	// router.HandleFunc("/commits", srv.loginRequired(srv.handleCommits))

	// API.
	router.PathPrefix("/api/").Handler(srv.loginOrTokenRequired(srv.api()))

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
	user, err := srv.getProfile(r)
	if err != nil {
		httpError(rw, r, err)
		return
	}

	// Read the template.
	t, err := srv.loadTemplates("home.html", "page_header.html", "page_footer.html")
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
		"Home",
		user,
		srv.relativePath("/auth/google/logout?next=") + url.QueryEscape("/"),
	}
	if err := t.Execute(&content, ctx); err != nil {
		httpError(rw, r, err)
		return
	}
	io.Copy(rw, &content)
}

func (srv *Server) handleLogin(rw http.ResponseWriter, r *http.Request) {
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
		srv.relativePath("/auth/google/login"),
	}
	if err := t.Execute(&content, ctx); err != nil {
		httpError(rw, r, err)
		return
	}
	io.Copy(rw, &content)
}

func (srv *Server) handleProfile(rw http.ResponseWriter, r *http.Request) {
	// Get the user record.
	user, err := srv.getProfile(r)
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
	}{
		srv.pathPrefix,
		"Profile",
		user,
	}
	if err := t.Execute(&content, ctx); err != nil {
		httpError(rw, r, err)
		return
	}
	io.Copy(rw, &content)
}

func (srv *Server) api() http.Handler {
	// API routing.
	api := &API{srv.store}

	router := mux.NewRouter()
	topRouter := mux.NewRouter()
	topRouter.PathPrefix("/v1").Handler(router)

	router.Path("/users/{userId}/generateToken").Methods("GET").HandlerFunc(api.GetGenerateToken)

	// Cover the whole API with token authentication.
	// TODO: Insert authentication middleware.
	n := negroni.New()
	n.UseHandler(topRouter)
	return n
}

func (srv *Server) getProfile(r *http.Request) (*common.User, error) {
	// Get session for the given HTTP request.
	var (
		session = sessions.GetSession(r)
		token   = noauth2.GetToken(r)
	)

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
		if err := marshalProfile(session, profile); err != nil {
			return nil, err
		}
	}

	// Fetch the user record from the store.
	user, err := srv.store.FindUserByEmail(profile.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return srv.createUser(profile)
	}
	return user, nil
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
			http.Redirect(rw, r, srv.relativePath("/login"), http.StatusTemporaryRedirect)
		} else {
			next.ServeHTTP(rw, r)
		}
	})
}

func (srv *Server) loginOrTokenRequired(next http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// Try the session token.
		token := noauth2.GetToken(r)
		if token != nil && token.Valid() {
			next.ServeHTTP(rw, r)
			return
		}

		// Try the access token.
		accessToken := r.Header.Get(TokenHeader)
		if accessToken != "" {
			user, err := srv.store.FindUserByToken(accessToken)
			if err != nil {
				httpError(rw, r, err)
				return
			}
			if user != nil {
				next.ServeHTTP(rw, r)
				return
			}
		}

		// Otherwise, unauthorized.
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
}

func (srv *Server) createUser(profile *userProfile) (*common.User, error) {
	user := &common.User{
		Name:  profile.Name,
		Email: profile.Email,
	}
	token, err := generateAccessToken()
	if err != nil {
		return nil, err
	}
	user.Token = token
	if err := srv.store.SaveUser(user); err != nil {
		return nil, err
	}
	return user, nil
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
