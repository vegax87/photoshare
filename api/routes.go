package api

import (
	"github.com/jmoiron/sqlx"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
	"net/http"
)

type HttpError struct {
	Status      int
	Description string
}

func (h HttpError) Error() string {
	if h.Description == "" {
		return http.StatusText(h.Status)
	}
	return h.Description
}

func httpError(status int, description string) HttpError {
	return HttpError{status, description}
}

type AppHandler func(c web.C, w http.ResponseWriter, r *http.Request) error

func (h AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(web.C{}, w, r)
	handleError(w, r, err)
}

func (h AppHandler) ServeHTTPC(c web.C, w http.ResponseWriter, r *http.Request) {
	err := h(c, w, r)
	handleError(w, r, err)
}

type AppContext struct {
	config     *AppConfig
	ds         DataStore
	fileMgr    FileManager
	sessionMgr SessionManager
	mailer     Mailer
}

func NewAppContext(config *AppConfig, db *sqlx.DB) (*AppContext, error) {

	ds := NewDataStore(db)
	fileMgr := NewFileManager(config)
	mailer := NewMailer(config)

	sessionMgr, err := NewSessionManager(config)
	if err != nil {
		return nil, err
	}

	a := &AppContext{
		config:     config,
		ds:         ds,
		fileMgr:    fileMgr,
		sessionMgr: sessionMgr,
		mailer:     mailer,
	}
	return a, nil
}

func GetRouter(config *AppConfig, db *sqlx.DB) (*web.Mux, error) {

	a, err := NewAppContext(config, db)
	if err != nil {
		return nil, err
	}
	r := web.New()

	r.Use(middleware.EnvInit)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AutomaticOptions)

	r.Get("/api/photos/", AppHandler(a.getPhotos))
	r.Post("/api/photos/", AppHandler(a.upload))
	r.Get("/api/photos/search", AppHandler(a.searchPhotos))
	r.Get("/api/photos/owner/:ownerID", AppHandler(a.photosByOwnerID))

	r.Get("/api/photos/:id", AppHandler(a.photoDetail))
	r.Delete("/api/photos/:id", AppHandler(a.deletePhoto))
	r.Patch("/api/photos/:id/title", AppHandler(a.editPhotoTitle))
	r.Patch("/api/photos/:id/tags", AppHandler(a.editPhotoTags))
	r.Patch("/api/photos/:id/upvote", AppHandler(a.voteUp))
	r.Patch("/api/photos/:id/downvote", AppHandler(a.voteDown))

	r.Get("/api/tags/", AppHandler(a.getTags))

	r.Get("/api/auth/", AppHandler(a.getSessionInfo))
	r.Post("/api/auth/", AppHandler(a.login))
	r.Delete("/api/auth/", AppHandler(a.logout))
	r.Post("/api/auth/signup", AppHandler(a.signup))
	r.Put("/api/auth/recoverpass", AppHandler(a.recoverPassword))
	r.Put("/api/auth/changepass", AppHandler(a.changePassword))

	r.Get("/feeds/", AppHandler(a.latestFeed))
	r.Get("/feeds/popular/", AppHandler(a.popularFeed))
	r.Get("/feeds/owner/:ownerID", AppHandler(a.ownerFeed))

	r.Handle("/api/messages/*", messageHandler)
	r.Handle("/*", http.FileServer(http.Dir(config.PublicDir)))
	return r, nil
}
