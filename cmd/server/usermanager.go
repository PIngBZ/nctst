package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

var (
	UserMgr = &UserManager{}
)

func init() {
	go UserMgr.daemon()
}

type UserInfo struct {
	ID       string
	Name     string
	Password string
	Hash     string
}

type UserManager struct {
	authCodes sync.Map
}

func (h *UserManager) CheckUserPassword(username, hash string) bool {
	var count int
	err := DB.QueryRow("select count(*) from userinfo where username=? and password=?", username, hash).Scan(&count)
	if err != nil {
		log.Printf("db query user error %s %+v\n", username, err)
		return false
	}
	return count != 0
}

func (h *UserManager) CheckAuthCode(username string, code int) bool {
	if c, ok := h.authCodes.Load(username); ok {
		return c.(int) == code
	}
	return false
}

func (h *UserManager) daemon() {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Throttle(30))
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(h.BasicAuth)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Route("/users", func(r chi.Router) {
		r.Get("/", h.ListUsers)

		r.Route("/{username}", func(r chi.Router) {
			r.Use(h.UserCtx)
			r.Get("/", h.GetUser)
			r.Put("/", h.UpdateUser)
			r.Delete("/", h.DeleteUser)
			r.Get("/authcode", h.GenerateAuthCode)
		})
	})

	http.ListenAndServe(config.AdminListen, r)
}

func (h *UserManager) BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Add("WWW-Authenticate", `Basic realm="Need Login"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !h.CheckUserPassword(user, pass) {
			time.Sleep(time.Second * 5)
			w.Header().Add("WWW-Authenticate", `Basic realm="Need Login"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

var UserContextKey = &nctst.ContextKey{Key: "user"}

func (h *UserManager) UserCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user *UserInfo
		var err error
		if username := chi.URLParam(r, "username"); username != "" {
			user, err = h.dbGetUser(username)
		} else {
			render.Render(w, r, nctst.ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, nctst.ErrNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *UserManager) dbGetUser(username string) (*UserInfo, error) {
	var id, name, pwd string
	if err := DB.QueryRow("select id, password from userinfo where username=?", username).Scan(&id, &name, &pwd); err != nil {
		return nil, err
	}
	user := &UserInfo{}
	user.ID = id
	user.Name = username
	user.Hash = pwd
	return user, nil
}

func (h *UserManager) ListUsers(w http.ResponseWriter, r *http.Request) {

}

func (h *UserManager) GetUser(w http.ResponseWriter, r *http.Request) {

}

func (h *UserManager) UpdateUser(w http.ResponseWriter, r *http.Request) {

}

func (h *UserManager) DeleteUser(w http.ResponseWriter, r *http.Request) {

}

func (h *UserManager) GenerateAuthCode(w http.ResponseWriter, r *http.Request) {
	//n := 1000 + rand.Intn(8999)
}
