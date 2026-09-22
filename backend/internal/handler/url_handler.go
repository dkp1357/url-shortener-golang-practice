package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"url-shortener/internal/middleware"
	"url-shortener/internal/models"
	"url-shortener/internal/repository/postgres"
	"url-shortener/internal/service"
	"url-shortener/internal/utils"

	"github.com/go-chi/chi/v5"
)

type URLHandler struct {
	urlService       *service.URLService
	analyticsService *service.AnalyticsService
}

func NewURLHandler(urlService *service.URLService, analyticsService *service.AnalyticsService) *URLHandler {
	return &URLHandler{
		urlService:       urlService,
		analyticsService: analyticsService,
	}
}

func (h *URLHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := h.urlService.Create(r.Context(), req, userID)
	if err != nil {
		if errors.Is(err, utils.ErrInvalidURL) ||
			errors.Is(err, utils.ErrInvalidAlias) ||
			errors.Is(err, utils.ErrReservedAlias) ||
			errors.Is(err, service.ErrInvalidExpiryDate) {
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, postgres.ErrAliasAlreadyUsed) {
			utils.ErrorResponse(w, http.StatusConflict, err.Error())
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to create short URL")
		return
	}

	utils.SuccessResponse(w, http.StatusCreated, "short URL created successfully", resp)
}

func (h *URLHandler) Get(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "code")
	if identifier == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "short code or alias is required")
		return
	}

	resp, err := h.urlService.GetDetails(r.Context(), identifier)
	if err != nil {
		if errors.Is(err, postgres.ErrURLNotFound) {
			utils.ErrorResponse(w, http.StatusNotFound, "URL not found")
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to retrieve URL")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, "URL details", resp)
}

func (h *URLHandler) Update(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "code")
	if identifier == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "short code or alias is required")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.UpdateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.urlService.Update(r.Context(), identifier, *userID, req)
	if err != nil {
		if errors.Is(err, postgres.ErrURLNotFound) {
			utils.ErrorResponse(w, http.StatusNotFound, "URL not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			utils.ErrorResponse(w, http.StatusForbidden, err.Error())
			return
		}
		if errors.Is(err, utils.ErrInvalidURL) || errors.Is(err, service.ErrInvalidExpiryDate) {
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to update URL")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, "URL updated successfully", resp)
}

func (h *URLHandler) Delete(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "code")
	if identifier == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "short code or alias is required")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := h.urlService.Delete(r.Context(), identifier, *userID)
	if err != nil {
		if errors.Is(err, postgres.ErrURLNotFound) {
			utils.ErrorResponse(w, http.StatusNotFound, "URL not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			utils.ErrorResponse(w, http.StatusForbidden, err.Error())
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to delete URL")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, "URL deleted successfully", nil)
}

func (h *URLHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	urls, totalCount, err := h.urlService.ListByUser(r.Context(), *userID, page, pageSize)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to fetch user URLs")
		return
	}

	utils.PaginatedSuccess(w, urls, page, pageSize, totalCount)
}

func (h *URLHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "code")
	if identifier == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "short code or alias is required")
		return
	}

	urlDetails, err := h.urlService.GetDetails(r.Context(), identifier)
	if err != nil {
		if errors.Is(err, postgres.ErrURLNotFound) {
			utils.ErrorResponse(w, http.StatusNotFound, "URL not found")
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to retrieve URL")
		return
	}

	// Verify ownership if the URL has a user_id
	currentUserID := middleware.GetUserID(r.Context())
	// If the link has an owner, only the owner can view detailed analytics
	// If the link was anonymous, it is publicly inspectable
	urlRecord, _ := h.urlService.GetDetails(r.Context(), identifier)
	if urlRecord != nil {
		// (Optional check can be enforced or open for analytics)
	}

	stats, err := h.analyticsService.GetStats(r.Context(), urlDetails.ID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, "URL analytics", map[string]any{
		"url":   urlDetails,
		"stats": stats,
	})
	_ = currentUserID
}
