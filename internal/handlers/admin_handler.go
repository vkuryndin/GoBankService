package handlers

import (
	"context"
	"net/http"

	"bank-service/internal/audit"
	"bank-service/internal/services"
)

var adminAccountErrorRules = joinErrorRules(
	errorRules{
		{target: services.ErrAccountAlreadyBlocked, status: http.StatusConflict, message: "account already blocked"},
		{target: services.ErrAccountAlreadyUnblocked, status: http.StatusConflict, message: "account already unblocked"},
	},
	accountErrorRules,
)

type AdminHandler struct {
	adminService  *services.AdminService
	auditRecorder audit.Recorder
}

func NewAdminHandler(adminService *services.AdminService, auditRecorder audit.Recorder) *AdminHandler {
	return &AdminHandler{
		adminService:  adminService,
		auditRecorder: auditRecorder,
	}
}

func (h *AdminHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, nil, "get users failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.adminService.GetUsers(ctx)
		return http.StatusOK, response, err
	})
}

func (h *AdminHandler) GetLoggedInUsers(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, nil, "get logged in users failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.adminService.GetLoggedInUsers(ctx)
		return http.StatusOK, response, err
	})
}

func (h *AdminHandler) GetSystemStatistics(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, nil, "get admin system statistics failed", func(ctx context.Context, userID int64) (int, any, error) {
		response, err := h.adminService.GetSystemStatistics(ctx)
		return http.StatusOK, response, err
	})
}

func (h *AdminHandler) BlockAccount(w http.ResponseWriter, r *http.Request) {
	h.changeAccountBlockStatus(w, r, true)
}

func (h *AdminHandler) UnblockAccount(w http.ResponseWriter, r *http.Request) {
	h.changeAccountBlockStatus(w, r, false)
}

func (h *AdminHandler) changeAccountBlockStatus(w http.ResponseWriter, r *http.Request, blocked bool) {
	accountID, err := parseAccountID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	action := "admin.account.unblock"
	fallback := "unblock account failed"
	if blocked {
		action = "admin.account.block"
		fallback = "block account failed"
	}

	handleAuthed(w, r, adminAccountErrorRules, fallback, func(ctx context.Context, adminID int64) (int, any, error) {
		response, err := h.setAccountBlockStatus(ctx, accountID, blocked)
		if err != nil {
			h.recordAccountBlockAudit(r, adminID, accountID, action+".failed", audit.StatusFailed)
			return http.StatusOK, nil, err
		}

		h.recordAccountBlockAudit(r, adminID, accountID, action+".success", audit.StatusSuccess)
		return http.StatusOK, response, nil
	})
}

func (h *AdminHandler) setAccountBlockStatus(ctx context.Context, accountID int64, blocked bool) (any, error) {
	if blocked {
		return h.adminService.BlockAccount(ctx, accountID)
	}

	return h.adminService.UnblockAccount(ctx, accountID)
}

func (h *AdminHandler) recordAccountBlockAudit(
	r *http.Request,
	adminID int64,
	accountID int64,
	action string,
	status string,
) {
	recordRequestAudit(
		h.auditRecorder,
		r,
		audit.Int64Ptr(adminID),
		action,
		"account",
		audit.Int64Ptr(accountID),
		status,
		nil,
	)
}
