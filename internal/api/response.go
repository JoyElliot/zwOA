package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/JoyElliot/zwOA/internal/modules/chat"
	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/gogf/gf/v2/net/ghttp"
)

func readJSON(r *ghttp.Request, value any) error {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || r.Header.Get("X-ZwOA-Request") != "1" {
		return organization.ErrInvalid
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	object := strings.TrimSpace(string(body))
	if !strings.HasPrefix(object, "{") {
		return organization.ErrInvalid
	}
	decoder := json.NewDecoder(strings.NewReader(object))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil || decoder.Decode(new(any)) != io.EOF {
		return organization.ErrInvalid
	}
	return nil
}

func writeResult(r *ghttp.Request, data any, err error) {
	if err != nil {
		writeError(r, err)
		return
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteJson(ghttp.DefaultHandlerResponse{Data: data})
}

func writeError(r *ghttp.Request, err error) {
	status := http.StatusInternalServerError
	message := err.Error()
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		status = http.StatusRequestEntityTooLarge
		message = "请求内容过大"
	case errors.Is(err, organization.ErrInvalid), errors.Is(err, organization.ErrPassword), errors.Is(err, chat.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, organization.ErrUnauthorized), errors.Is(err, organization.ErrCredentials):
		status = http.StatusUnauthorized
	case errors.Is(err, organization.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, organization.ErrNotFound), errors.Is(err, chat.ErrMissing):
		status = http.StatusNotFound
	case errors.Is(err, organization.ErrConflict), errors.Is(err, chat.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, organization.ErrStale):
		status = http.StatusPreconditionFailed
	case errors.Is(err, organization.ErrTooMany):
		status = http.StatusTooManyRequests
		r.Response.Header().Set("Retry-After", "60")
	default:
		log.Printf("业务接口失败：%v", err)
		message = "服务暂时不可用"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJson(ghttp.DefaultHandlerResponse{Code: status, Message: message})
}

func recordID(r *ghttp.Request) (int64, error) {
	id, err := strconv.ParseInt(r.GetRouter("id").String(), 10, 64)
	if err != nil || id <= 0 {
		return 0, organization.ErrInvalid
	}
	return id, nil
}
