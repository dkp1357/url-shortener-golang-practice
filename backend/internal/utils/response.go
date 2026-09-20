package utils

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type PaginatedResponse struct {
	Success    bool  `json:"success"`
	Data       any   `json:"data"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalCount int64 `json:"total_count"`
	TotalPages int   `json:"total_pages"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func SuccessResponse(w http.ResponseWriter, status int, message string, data interface{}) {
	WriteJSON(w, status, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(w http.ResponseWriter, status int, errMsg string) {
	WriteJSON(w, status, Response{
		Success: false,
		Error:   errMsg,
	})
}

func PaginatedSuccess(w http.ResponseWriter, data any, page, pageSize int, totalCount int64) {
	totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	WriteJSON(w, http.StatusOK, PaginatedResponse{
		Success:    true,
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	})
}
