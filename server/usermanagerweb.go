package main

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/PIngBZ/nctst"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

var (
	LoginUserContextKey  = &nctst.ContextKey{Key: "login_user_context_key"}
	TargetUserContextKey = &nctst.ContextKey{Key: "user_context_key"}
)

func init() {
	go UserMgr.daemon()
}

func (h *UserManager) daemon() {
	r := chi.NewRouter()

	r.Use(middleware.Throttle(30))
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(h.basicAuth)

	r.Get("/initdev", h.initAuthDevice)
	r.Get("/authcode", h.generateAuthCode)
	r.Get("/exit", h.exit)

	r.Route("/users", func(r chi.Router) {
		r.Get("/", h.listUsers)
		r.Get("/add", h.addUser)
		r.Post("/commit", h.commitUser)

		r.Route("/{username}", func(r chi.Router) {
			r.Use(h.userCtx)
			r.Get("/del", h.deleteUser)
			r.Get("/admin", h.changeAdmin)
			r.Get("/changepwd", h.changePwd)
			r.Post("/commitpwd", h.commitPwd)
			r.Get("/proxy", h.changeProxy)
		})
	})

	http.ListenAndServe(config.AdminListen, r)
}

func (h *UserManager) basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Add("WWW-Authenticate", `Basic realm="Need Login"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		user, err := h.GetUser(name)
		if err != nil {
			log.Printf("basicAuth GetUser error %+v\n", err)
			time.Sleep(time.Second * 2)
			render.Render(w, r, ErrForbiddenErrLogin)
			return
		}

		if nctst.HashPassword(name, pass) != user.Hash {
			time.Sleep(time.Second * 5)
			render.Render(w, r, ErrForbiddenErrLogin)
			return
		}

		DB.Exec("update userinfo set lasttime=now() where id=?", user.ID)

		ctx := context.WithValue(r.Context(), LoginUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *UserManager) isAdmin(r *http.Request) bool {
	user, success := r.Context().Value(LoginUserContextKey).(*UserInfo)
	if !success {
		return false
	}
	return user.Admin
}

func (h *UserManager) userCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user *UserInfo
		var err error
		if username := chi.URLParam(r, "username"); username != "" {
			user, err = h.GetUser(username)
		} else {
			render.Render(w, r, ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, ErrNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), TargetUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *UserManager) GetUser(username string) (*UserInfo, error) {
	var id, realName, hash, session string
	var admin, status, proxy int
	var lastTime, createTime time.Time
	cmd := "select id,realname,password,admin,session,lasttime,createtime,status,proxy from userinfo where username=?"
	if err := DB.QueryRow(cmd, username).Scan(&id, &realName, &hash, &admin, &session, &lastTime, &createTime, &status, &proxy); err != nil {
		return nil, err
	}
	user := &UserInfo{}
	user.ID = id
	user.UserName = username
	user.RealName = realName
	user.Hash = hash
	user.Admin = admin == 1
	user.Session = session
	user.LastTime = lastTime
	user.CreateTime = createTime
	user.Status = UserStatus(status)
	user.Proxy = proxy == 1

	if c, loaded := h.authCodes.Load(username); loaded {
		user.CodeInfo = c.(*CodeInfo)
		user.CodeInfo.Seconds = int(time.Until(user.CodeInfo.Time.Add(time.Second*60)) / time.Second)
	}

	return user, nil
}

type ListUserRenderData struct {
	Me           *UserInfo
	List         []*UserInfo
	InitCode     int32
	InitCodeTime int
}

func (h *UserManager) getDataCounts(t int) (map[string]nctst.Pair[uint64, uint64], error) {
	w := "%Y-%m-%d-%H" // hour
	if t == 1 {
		w = "%Y-%m-%d)" // day
	} else if t == 2 {
		w = "%Y-%W" // week
	} else if t == 3 {
		w = "%Y-%m" // month
	}
	rows, err := DB.Query("select username,sum(send),sum(receive) from datacount where strftime('" + w + "',savetime)=strftime('" + w + "','now') group by username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userName string
	var send, receive uint64
	data := make(map[string]nctst.Pair[uint64, uint64])
	for rows.Next() {
		if err = rows.Scan(&userName, &send, &receive); err != nil {
			return nil, err
		}
		data[userName] = nctst.Pair[uint64, uint64]{First: send, Second: receive}
	}

	return data, nil
}

func (h *UserManager) listUsers(w http.ResponseWriter, r *http.Request) {
	login, _ := r.Context().Value(LoginUserContextKey).(*UserInfo)

	hourCounts, err := h.getDataCounts(0)
	if err != nil {
		hourCounts = make(map[string]nctst.Pair[uint64, uint64])
	}
	dayCounts, err := h.getDataCounts(1)
	if err != nil {
		dayCounts = make(map[string]nctst.Pair[uint64, uint64])
	}
	weekCounts, err := h.getDataCounts(2)
	if err != nil {
		weekCounts = make(map[string]nctst.Pair[uint64, uint64])
	}
	monthCounts, err := h.getDataCounts(3)
	if err != nil {
		monthCounts = make(map[string]nctst.Pair[uint64, uint64])
	}

	var id, userName, realName, hash string
	var admin, status, proxy int
	var lastTime, createTime time.Time
	cmd := "select id,username,realname,password,admin,lasttime,createtime,status,proxy from userinfo"
	if !login.Admin {
		cmd += " where id=" + login.ID
	} else {
		cmd += " order by id"
	}

	rows, err := DB.Query(cmd)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	defer rows.Close()

	users := make([]*UserInfo, 0)
	for rows.Next() {
		if err = rows.Scan(&id, &userName, &realName, &hash, &admin, &lastTime, &createTime, &status, &proxy); err != nil {
			render.Render(w, r, ErrInternal(err))
			return
		}
		user := &UserInfo{}
		user.ID = id
		user.UserName = userName
		user.RealName = realName
		user.Hash = hash
		user.Admin = admin == 1
		user.LastTime = lastTime
		user.CreateTime = createTime
		user.Status = UserStatus(status)
		user.Proxy = proxy == 1

		if dc, ok := hourCounts[userName]; ok {
			user.TrafficHour.Send = dc.First
			user.TrafficHour.Receive = dc.Second
		}
		if dc, ok := dayCounts[userName]; ok {
			user.TrafficDay.Send = dc.First
			user.TrafficDay.Receive = dc.Second
		}
		if dc, ok := weekCounts[userName]; ok {
			user.TrafficWeek.Send = dc.First
			user.TrafficWeek.Receive = dc.Second
		}
		if dc, ok := monthCounts[userName]; ok {
			user.TrafficMonth.Send = dc.First
			user.TrafficMonth.Receive = dc.Second
		}

		if c, loaded := h.authCodes.Load(userName); loaded {
			user.CodeInfo = c.(*CodeInfo)
			user.CodeInfo.Seconds = int(time.Until(user.CodeInfo.Time.Add(time.Second*60)) / time.Second)
		}

		users = append(users, user)
	}

	t, err := template.ParseFiles("html/listusers.html")
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	initCodeValid := false
	if ta := h.initCodeTime.Load(); ta != nil {
		t := ta.(time.Time)
		initCodeValid = t.Add(time.Minute * 5).After(time.Now())
	}
	if !initCodeValid {
		h.initCode.Store(uint32(1000 + rand.Intn(9000)))
		h.initCodeTime.Store(time.Now())
	}

	var data *ListUserRenderData
	if login.Admin {
		sec := time.Until(h.initCodeTime.Load().(time.Time).Add(time.Minute*5)) / time.Second
		data = &ListUserRenderData{Me: login, List: users, InitCode: int32(h.initCode.Load()), InitCodeTime: int(sec)}
	} else {
		data = &ListUserRenderData{Me: login, List: users}
	}

	err = t.Execute(w, data)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
}

func (h *UserManager) addUser(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		render.Render(w, r, ErrForbiddenNeedAdmin)
		return
	}

	t, err := template.ParseFiles("html/adduser.html")
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	err = t.Execute(w, nil)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
}

func (h *UserManager) commitUser(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		render.Render(w, r, ErrForbiddenNeedAdmin)
		return
	}

	r.ParseForm()

	userName := r.Form.Get("username")
	pwd := r.Form.Get("password")
	realName := r.Form.Get("realname")

	if userName == "" || pwd == "" || realName == "" {
		render.Render(w, r, ErrInvalidRequest(errors.New("params error")))
		return
	}

	pwd = nctst.HashPassword(userName, pwd)

	adminS := r.Form.Get("admin")
	admin := 0
	if adminS == "1" {
		admin = 1
	}

	cmd := "insert into userinfo(username,realname,password,admin) values(?,?,?,?)"
	_, err := DB.Exec(cmd, userName, realName, pwd, admin)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *UserManager) deleteUser(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		render.Render(w, r, ErrForbiddenNeedAdmin)
		return
	}

	user, _ := r.Context().Value(TargetUserContextKey).(*UserInfo)
	_, err := DB.Exec("delete from userinfo where id=?", user.ID)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *UserManager) changeAdmin(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		render.Render(w, r, ErrForbiddenNeedAdmin)
		return
	}

	user, _ := r.Context().Value(TargetUserContextKey).(*UserInfo)

	if user.UserName == "admin" {
		render.Render(w, r, ErrForbidden)
		return
	}

	toAdmin := 1
	if user.Admin {
		toAdmin = 0
	}
	_, err := DB.Exec("update userinfo set admin=? where id=?", toAdmin, user.ID)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *UserManager) changePwd(w http.ResponseWriter, r *http.Request) {
	login, _ := r.Context().Value(LoginUserContextKey).(*UserInfo)
	target, _ := r.Context().Value(TargetUserContextKey).(*UserInfo)

	if !h.isAdmin(r) && login.ID != target.ID {
		render.Render(w, r, ErrForbiddenNeedAdmin)
		return
	}

	t, err := template.ParseFiles("html/changepwd.html")
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	err = t.Execute(w, target)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
}

func (h *UserManager) commitPwd(w http.ResponseWriter, r *http.Request) {
	login, _ := r.Context().Value(LoginUserContextKey).(*UserInfo)
	target, _ := r.Context().Value(TargetUserContextKey).(*UserInfo)

	if !h.isAdmin(r) && login.ID != target.ID {
		render.Render(w, r, ErrForbiddenNeedAdmin)
		return
	}

	r.ParseForm()
	newPwd := r.Form.Get("password")
	if len(newPwd) < 6 {
		render.Render(w, r, ErrForbiddenPwdTooShort)
		return
	}

	_, err := DB.Exec("update userinfo set password=? where id=?", nctst.HashPassword(target.UserName, newPwd), target.ID)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *UserManager) changeProxy(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		render.Render(w, r, ErrForbiddenNeedAdmin)
		return
	}

	user, _ := r.Context().Value(TargetUserContextKey).(*UserInfo)

	toOpenProxy := 1
	if user.Proxy {
		toOpenProxy = 0
	}
	_, err := DB.Exec("update userinfo set proxy=? where id=?", toOpenProxy, user.ID)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *UserManager) generateAuthCode(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(LoginUserContextKey).(*UserInfo)

	r.ParseForm()

	session := r.Form.Get("session")
	if session != user.Session {
		render.Render(w, r, ErrForbiddenNeedInit)
		return
	}

	newCode := &CodeInfo{Code: 1000 + rand.Intn(9000), Time: time.Now()}
	if c, loaded := h.authCodes.LoadOrStore(user.UserName, newCode); loaded {
		info := c.(*CodeInfo)
		if info.Time.Add(time.Second * 50).Before(time.Now()) {
			h.authCodes.Store(user.UserName, newCode)
			info = newCode
		}

		seconds := int(time.Until(info.Time.Add(time.Second*60)) / time.Second)
		WriteResponse(w, &CodeResponse{AuthCode: info.Code, Seconds: seconds})
	} else {
		WriteResponse(w, &CodeResponse{AuthCode: newCode.Code, Seconds: 60})
	}
}

func (h *UserManager) initAuthDevice(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(LoginUserContextKey).(*UserInfo)
	r.ParseForm()

	initCodeS := r.Form.Get("code")
	if initCodeS == "" {
		render.Render(w, r, ErrInvalidRequest(errors.New("error params")))
		return
	}
	initCode, err := strconv.Atoi(initCodeS)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if ta := h.initCodeTime.Load(); ta == nil {
		render.Render(w, r, ErrForbidden)
		return
	} else {
		t := ta.(time.Time)
		if !t.Add(time.Minute * 5).After(time.Now()) {
			render.Render(w, r, ErrForbidden)
			return
		}

		if initCode != int(h.initCode.Load()) {
			render.Render(w, r, ErrForbidden)
			return
		}

		session := uuid.NewString()
		if _, err := DB.Exec("update userinfo set session=? where id=?", session, user.ID); err != nil {
			render.Render(w, r, ErrInternal(err))
			return
		}

		WriteResponse(w, &InitResponse{Session: session})
	}
}

func (h *UserManager) exit(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "exit success", http.StatusUnauthorized)
}
