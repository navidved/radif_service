package auth

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/radif/service/internal/response"
)

// iranPhoneRegex matches valid Iranian mobile numbers (09XXXXXXXXX).
var iranPhoneRegex = regexp.MustCompile(`^09[0-9]{9}$`)

// Handler holds HTTP handlers for auth endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new auth Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type sendOTPRequest struct {
	Phone string `json:"phone" example:"09121234567"`
}

type verifyOTPRequest struct {
	Phone string `json:"phone" example:"09121234567"`
	Code  string `json:"code"  example:"12345"`
}

type registerRequest struct {
	Phone       string `json:"phone"       example:"09121234567"`
	AccountType string `json:"accountType" example:"personal"`
}

type otpSuccessData struct {
	Success bool `json:"success" example:"true"`
}

type verifyOTPData struct {
	IsNewUser bool   `json:"isNewUser" example:"true"`
	Token     string `json:"token,omitempty" example:"eyJhbGci..."`
}

type registerData struct {
	Token string   `json:"token" example:"eyJhbGci..."`
	User  userBody `json:"user"`
}

type userBody struct {
	ID          string `json:"id"          example:"e7eedc79-0707-4fe4-8734-526b7ef13a7b"`
	Phone       string `json:"phone"       example:"09121234567"`
	AccountType string `json:"accountType" example:"personal"`
	CreatedAt   string `json:"createdAt"   example:"2026-02-27T14:48:34Z"`
	UpdatedAt   string `json:"updatedAt"   example:"2026-02-27T14:48:34Z"`
}

// SendOTP godoc
//
//	@Summary		Send OTP
//	@Description	Generate and send a 5-digit OTP to the given Iranian mobile number. In development the code is printed to server logs.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		sendOTPRequest					true	"Phone number"
//	@Success		200		{object}	response.Envelope{data=otpSuccessData}
//	@Failure		400		{object}	response.Envelope
//	@Failure		500		{object}	response.Envelope
//	@Router			/auth/otp/send [post]
func (h *Handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req sendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if !iranPhoneRegex.MatchString(req.Phone) {
		response.BadRequest(w, "invalid phone number format")
		return
	}

	if err := h.svc.SendOTP(r.Context(), req.Phone); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]bool{"success": true})
}

// VerifyOTP godoc
//
//	@Summary		Verify OTP
//	@Description	Validate the OTP code. Returns isNewUser=true for first-time users (no token yet). Returns a JWT token for existing users immediately.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		verifyOTPRequest				true	"Phone and OTP code"
//	@Success		200		{object}	response.Envelope{data=verifyOTPData}
//	@Failure		400		{object}	response.Envelope
//	@Failure		500		{object}	response.Envelope
//	@Router			/auth/otp/verify [post]
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req verifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if !iranPhoneRegex.MatchString(req.Phone) {
		response.BadRequest(w, "invalid phone number format")
		return
	}
	if len(req.Code) != 5 {
		response.BadRequest(w, "OTP code must be exactly 5 digits")
		return
	}

	result, err := h.svc.VerifyOTP(r.Context(), req.Phone, req.Code)
	if err == ErrInvalidOTP {
		response.BadRequest(w, "invalid or expired OTP")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	data := map[string]interface{}{
		"isNewUser": result.IsNewUser,
	}
	if result.Token != "" {
		data["token"] = result.Token
	}
	response.OK(w, data)
}

// ResendOTP godoc
//
//	@Summary		Resend OTP
//	@Description	Invalidate the current OTP and issue a new one with a fresh 2-minute TTL.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		sendOTPRequest					true	"Phone number"
//	@Success		200		{object}	response.Envelope{data=otpSuccessData}
//	@Failure		400		{object}	response.Envelope
//	@Failure		500		{object}	response.Envelope
//	@Router			/auth/otp/resend [post]
func (h *Handler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var req sendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if !iranPhoneRegex.MatchString(req.Phone) {
		response.BadRequest(w, "invalid phone number format")
		return
	}

	if err := h.svc.SendOTP(r.Context(), req.Phone); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]bool{"success": true})
}

// Register godoc
//
//	@Summary		Register new user
//	@Description	Create a new user account with the specified account type. Issues a JWT token on success. Idempotent: calling again with the same phone returns a fresh token.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		registerRequest					true	"Registration details"
//	@Success		201		{object}	response.Envelope{data=registerData}
//	@Failure		400		{object}	response.Envelope
//	@Failure		500		{object}	response.Envelope
//	@Router			/auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if !iranPhoneRegex.MatchString(req.Phone) {
		response.BadRequest(w, "invalid phone number format")
		return
	}

	validTypes := map[string]bool{
		"personal": true,
		"children": true,
		"business": true,
	}
	if !validTypes[req.AccountType] {
		response.BadRequest(w, "accountType must be one of: personal, children, business")
		return
	}

	token, u, err := h.svc.Register(r.Context(), req.Phone, req.AccountType)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.Created(w, map[string]interface{}{
		"token": token,
		"user":  u,
	})
}
