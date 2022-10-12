package main

import (
	"net/http"

	"github.com/PIngBZ/nctst"
	"github.com/go-chi/render"
)

const (
	AppStatusCodeErrLogin = 1001
	AppStatusCodeNeedInit = 1002
)

type ErrResponse struct {
	Err            error `json:"-"`
	HTTPStatusCode int   `json:"code"`

	StatusText string `json:"status"`
	AppCode    int64  `json:"appcode,omitempty"`
	ErrorText  string `json:"error,omitempty"`
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 422,
		StatusText:     "Error rendering response.",
		ErrorText:      err.Error(),
	}
}

func ErrInternal(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 500,
		StatusText:     "Internal server error.",
		ErrorText:      err.Error(),
	}
}

var ErrNotFound = &ErrResponse{HTTPStatusCode: http.StatusNotFound, StatusText: http.StatusText(http.StatusNotFound)}
var ErrForbidden = &ErrResponse{HTTPStatusCode: http.StatusForbidden, StatusText: http.StatusText(http.StatusForbidden)}
var ErrForbiddenErrLogin = &ErrResponse{HTTPStatusCode: http.StatusForbidden, AppCode: AppStatusCodeErrLogin, StatusText: http.StatusText(http.StatusForbidden)}
var ErrForbiddenNeedInit = &ErrResponse{HTTPStatusCode: http.StatusForbidden, AppCode: AppStatusCodeNeedInit, StatusText: http.StatusText(http.StatusForbidden)}

type APIResponseCode int

const (
	APIResponseCode_Success = iota
)

type APIResponse struct {
	Code       APIResponseCode `json:"code"`
	StatusText string          `json:"status"`
	Data       interface{}     `json:"data"`
}

type InitResponse struct {
	Session string `json:"session"`
}

type CodeResponse struct {
	AuthCode int `json:"authcode"`
	Seconds  int `json:"seconds"`
}

func WriteResponse(w http.ResponseWriter, data interface{}) {
	resp := APIResponse{Code: APIResponseCode_Success, StatusText: "success", Data: data}
	if s, err := nctst.ToJson(resp); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	} else {
		w.Write([]byte(s))
	}
}
