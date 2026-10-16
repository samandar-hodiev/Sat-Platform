package profile

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
)

// My Profile routes (2.10): PATCH /me/profile, DELETE /me/account. docs/API.md → My Profile.

const (
	deleteAccountLimit  = 10
	deleteAccountWindow = 10 * time.Minute
)

// updateRequest: absent (or null) fields are not changed; phone "" removes the phone.
type updateRequest struct {
	FirstName   *string `json:"first_name" binding:"omitempty,max=400"`
	LastName    *string `json:"last_name" binding:"omitempty,max=400"`
	Phone       *string `json:"phone" binding:"omitempty,max=64"`
	DateOfBirth *string `json:"date_of_birth" binding:"omitempty,max=32"`
	Timezone    *string `json:"timezone" binding:"omitempty,max=64"`
}

func (h *Handler) update(c *gin.Context) {
	var req updateRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	v, err := h.svc.Update(c.Request.Context(), auth.CurrentUser(c).UserID, UpdateInput(req))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(v))
}

type deleteAccountRequest struct {
	Confirmation string `json:"confirmation" binding:"max=32"`
	auth.ReauthRequest
}

func (h *Handler) deleteAccount(c *gin.Context) {
	var req deleteAccountRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	in := DeleteAccountInput{Confirmation: req.Confirmation, Reauth: req.Reauth()}
	if err := h.svc.DeleteAccount(c.Request.Context(), auth.CurrentUser(c).UserID, in); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}
