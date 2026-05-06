package handlers

import (
	"errors"
	"net/http"

	"bank-service/internal/services"
)

type AdminHandler struct {
	adminService *services.AdminService
}

func NewAdminHandler(adminService *services.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

func (h *AdminHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	response, err := h.adminService.GetUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get users failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AdminHandler) GetLoggedInUsers(w http.ResponseWriter, r *http.Request) {
	response, err := h.adminService.GetLoggedInUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get logged in users failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AdminHandler) BlockAccount(w http.ResponseWriter, r *http.Request) {
	accountID, err := parseAccountID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	response, err := h.adminService.BlockAccount(r.Context(), accountID)
	if err != nil {
		if errors.Is(err, services.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}

		if errors.Is(err, services.ErrAccountAlreadyBlocked) {
			writeError(w, http.StatusConflict, "account already blocked")
			return
		}

		writeError(w, http.StatusInternalServerError, "block account failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AdminHandler) UnblockAccount(w http.ResponseWriter, r *http.Request) {
	accountID, err := parseAccountID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	response, err := h.adminService.UnblockAccount(r.Context(), accountID)
	if err != nil {
		if errors.Is(err, services.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}

		if errors.Is(err, services.ErrAccountAlreadyUnblocked) {
			writeError(w, http.StatusConflict, "account already unblocked")
			return
		}

		writeError(w, http.StatusInternalServerError, "unblock account failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}
