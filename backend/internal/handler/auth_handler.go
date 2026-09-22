package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"url-shortener/internal/middleware"
	"url-shortener/internal/models"
	"url-shortener/internal/repository/postgres"
	"url-shortener/internal/service"
	"url-shortener/internal/utils"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authService.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, postgres.ErrUserAlreadyExists) {
			utils.ErrorResponse(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidUsername) || errors.Is(err, service.ErrPasswordTooWeak) {
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	utils.SuccessResponse(w, http.StatusCreated, "user registered successfully", resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCreds) {
			utils.ErrorResponse(w, http.StatusUnauthorized, err.Error())
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to login")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, "login successful", resp)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	email := middleware.GetUsername(r.Context())

	if userID == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, "user profile", map[string]any{
		"id":    userID,
		"email": email,
	})
}
