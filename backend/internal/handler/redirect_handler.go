package handler

import (
	"errors"
	"net/http"
	"strings"

	"url-shortener/internal/repository/postgres"
	"url-shortener/internal/service"
	"url-shortener/internal/utils"

	"github.com/go-chi/chi/v5"
)

type RedirectHandler struct {
	urlService *service.URLService
}

func NewRedirectHandler(urlService *service.URLService) *RedirectHandler {
	return &RedirectHandler{urlService: urlService}
}

func (h *RedirectHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		http.NotFound(w, r)
		return
	}

	referrer := r.Referer()
	userAgent := r.UserAgent()
	ip := getIP(r)

	targetURL, err := h.urlService.GetAndTrack(r.Context(), code, referrer, userAgent, ip)
	if err != nil {
		if errors.Is(err, service.ErrURLExpired) {
			utils.ErrorResponse(w, http.StatusGone, "this short link has expired")
			return
		}
		if errors.Is(err, service.ErrURLInactive) {
			utils.ErrorResponse(w, http.StatusGone, "this short link has been disabled")
			return
		}
		if errors.Is(err, postgres.ErrURLNotFound) {
			utils.ErrorResponse(w, http.StatusNotFound, "short link not found")
			return
		}

		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to redirect")
		return
	}

	// 302 Found redirect preserves tracking without browser permanent caching
	http.Redirect(w, r, targetURL, http.StatusFound)
}

func getIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	return r.RemoteAddr
}
