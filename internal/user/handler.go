package user

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/radif/service/internal/middleware"
	"github.com/radif/service/internal/response"
	"github.com/radif/service/internal/storage"
)

const maxAvatarBytes = 5 << 20 // 5 MB

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// Handler holds HTTP handlers for user-related endpoints.
type Handler struct {
	svc   *Service
	store storage.Storage
}

// NewHandler creates a new user Handler.
func NewHandler(svc *Service, store storage.Storage) *Handler {
	return &Handler{svc: svc, store: store}
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

	h.populateAvatarURL(u)
	response.OK(w, u)
}

// UpdateProfile godoc
//
//	@Summary		Update profile
//	@Description	Partially update the authenticated user's profile (username, fullName, bio).
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		updateProfileRequest			true	"Profile fields to update"
//	@Success		200		{object}	response.Envelope{data=User}
//	@Failure		400		{object}	response.Envelope
//	@Failure		401		{object}	response.Envelope
//	@Failure		409		{object}	response.Envelope
//	@Failure		500		{object}	response.Envelope
//	@Router			/users/me [patch]
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Username != nil && *req.Username != "" {
		if !usernameRegex.MatchString(*req.Username) {
			response.BadRequest(w, "username may only contain letters, digits, and underscores")
			return
		}
		if len(*req.Username) > 50 {
			response.BadRequest(w, "username must be 50 characters or fewer")
			return
		}
	}

	if req.Bio != nil && len(*req.Bio) > 160 {
		response.BadRequest(w, "bio must be 160 characters or fewer")
		return
	}

	u, err := h.svc.UpdateProfile(r.Context(), userID, UpdateProfileParams{
		Username:      req.Username,
		FullName:      req.FullName,
		Bio:           req.Bio,
		BusinessPhone: req.BusinessPhone,
		Address:       req.Address,
	})
	if err != nil {
		if h.svc.IsUsernameTaken(err) {
			response.Conflict(w, "username is already taken")
			return
		}
		if h.svc.IsNotFound(err) {
			response.NotFound(w, "user not found")
			return
		}
		response.InternalError(w)
		return
	}

	h.populateAvatarURL(u)
	response.OK(w, u)
}

// UploadAvatar godoc
//
//	@Summary		Upload avatar
//	@Description	Upload a profile picture (JPEG/PNG/WebP/GIF, max 5 MB). Stores the file in object storage and saves the key.
//	@Tags			users
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			avatar	formData	file	true	"Image file"
//	@Success		200		{object}	response.Envelope{data=avatarUploadResponse}
//	@Failure		400		{object}	response.Envelope
//	@Failure		401		{object}	response.Envelope
//	@Failure		500		{object}	response.Envelope
//	@Router			/users/me/avatar [post]
func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		response.Unauthorized(w, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarBytes+1024)
	if err := r.ParseMultipartForm(maxAvatarBytes); err != nil {
		response.BadRequest(w, "file too large or invalid multipart form (max 5 MB)")
		return
	}

	file, _, err := r.FormFile("avatar")
	if err != nil {
		response.BadRequest(w, "field \"avatar\" is required")
		return
	}
	defer file.Close()

	// Read first 512 bytes to detect the actual content type.
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		response.InternalError(w)
		return
	}

	contentType := http.DetectContentType(buf[:n])
	ext, allowed := allowedImageTypes[contentType]
	if !allowed {
		response.BadRequest(w, "only JPEG, PNG, WebP, and GIF images are allowed")
		return
	}

	// Re-assemble the full stream: the 512 bytes we already read + the remainder.
	fullReader := io.MultiReader(bytes.NewReader(buf[:n]), file)

	key, err := generateStorageKey(userID, ext)
	if err != nil {
		response.InternalError(w)
		return
	}

	if err := h.store.Upload(r.Context(), key, fullReader, -1, contentType); err != nil {
		response.InternalError(w)
		return
	}

	if _, err := h.svc.UpdateAvatarKey(r.Context(), userID, key); err != nil {
		response.InternalError(w)
		return
	}

	avatarURL := h.store.PublicURL(key)
	response.OK(w, avatarUploadResponse{AvatarURL: avatarURL})
}

// populateAvatarURL attaches the public URL to the user struct when an avatar key is present.
func (h *Handler) populateAvatarURL(u *User) {
	if u.AvatarKey != nil && *u.AvatarKey != "" {
		url := h.store.PublicURL(*u.AvatarKey)
		u.AvatarURL = &url
	}
}

// generateStorageKey creates a collision-resistant object key for a user's avatar.
// Format: "{userID}/{16-byte-hex}{ext}"
func generateStorageKey(userID, ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	return fmt.Sprintf("%s/%x%s", userID, b, ext), nil
}

// CheckUsername godoc
//
//	@Summary		Check username availability
//	@Description	Returns whether the given username is available (not yet taken). Requires authentication.
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			username	query		string	true	"Username to check"
//	@Success		200			{object}	response.Envelope{data=usernameCheckResponse}
//	@Failure		400			{object}	response.Envelope
//	@Failure		401			{object}	response.Envelope
//	@Failure		500			{object}	response.Envelope
//	@Router			/users/username-check [get]
func (h *Handler) CheckUsername(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		response.BadRequest(w, "username query parameter is required")
		return
	}
	if !usernameRegex.MatchString(username) {
		response.BadRequest(w, "username may only contain letters, digits, and underscores")
		return
	}
	if len(username) > 50 {
		response.BadRequest(w, "username must be 50 characters or fewer")
		return
	}

	available, err := h.svc.UsernameAvailable(r.Context(), username)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, usernameCheckResponse{Available: available})
}

type updateProfileRequest struct {
	Username      *string `json:"username"`
	FullName      *string `json:"fullName"`
	Bio           *string `json:"bio"`
	BusinessPhone *string `json:"businessPhone"`
	Address       *string `json:"address"`
}

type avatarUploadResponse struct {
	AvatarURL string `json:"avatarUrl"`
}

type usernameCheckResponse struct {
	Available bool `json:"available"`
}
