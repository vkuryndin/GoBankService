package handlers

import (
	"context"
	"net/http"

	"bank-service/internal/audit"
	"bank-service/internal/services"
)

var adminAccountErrorRules = errorRules{
	{target: services.ErrAccountNotFound, statusCode: http.StatusNotFound, message: "account not found"},
	{target: services.ErrAccountAlreadyBlocked, statusCode: http.StatusConflict, message: "account already blocked"},
	{target: services.ErrAccountAlreadyUnblocked, statusCode: http.StatusConflict, message: "account already unblocked"},
}

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

	adminID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	action := "admin.account.unblock"
	fallback := "unblock account failed"
	operation := func(ctx context.Context) (any, error) {
		return h.adminService.UnblockAccount(ctx, accountID)
	}

	if blocked {
		action = "admin.account.block"
		fallback = "block account failed"
		operation = func(ctx context.Context) (any, error) {
			return h.adminService.BlockAccount(ctx, accountID)
		}
	}

	response, err := operation(r.Context())
	if err != nil {
		recordRequestAudit(
			h.auditRecorder,
			r,
			audit.Int64Ptr(adminID),
			action+".failed",
			"account",
			audit.Int64Ptr(accountID),
			audit.StatusFailed,
			nil,
		)
		writeMappedError(w, err, adminAccountErrorRules, fallback)
		return
	}

	recordRequestAudit(
		h.auditRecorder,
		r,
		audit.Int64Ptr(adminID),
		action+".success",
		"account",
		audit.Int64Ptr(accountID),
		audit.StatusSuccess,
		nil,
	)
	writeJSON(w, http.StatusOK, response)
}
