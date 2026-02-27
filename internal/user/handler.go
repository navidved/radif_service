package user

import (
	"net/http"

	"github.com/radif/service/internal/middleware"
	"github.com/radif/service/internal/response"
)

// Handler holds HTTP handlers for user-related endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new user Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetMe godoc
//
//	@Summary		Get current user
//	@Description	Returns the profile of the currently authenticated user.
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Envelope{data=User}
//	@Failure		401	{object}	response.Envelope
//	@Failure		404	{object}	response.Envelope
//	@Failure		500	{object}	response.Envelope
//	@Router			/users/me [get]
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		response.Unauthorized(w, "unauthorized")
		return
	}

	u, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		if h.svc.IsNotFound(err) {
			response.NotFound(w, "user not found")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, u)
}
