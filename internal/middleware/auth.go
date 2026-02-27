package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/radif/service/internal/response"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

// UserIDKey is the context key for the authenticated user's ID.
const UserIDKey contextKey = "userID"

// UserPhoneKey is the context key for the authenticated user's phone.
const UserPhoneKey contextKey = "userPhone"

// UserAccountTypeKey is the context key for the authenticated user's account type.
const UserAccountTypeKey contextKey = "userAccountType"

// RequireAuth returns middleware that validates a Bearer JWT and injects
// user claims into the request context.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "authorization header required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Unauthorized(w, "invalid authorization header format")
				return
			}

			token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				response.Unauthorized(w, "invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				response.Unauthorized(w, "invalid token claims")
				return
			}

			userID, _ := claims["sub"].(string)
			phone, _ := claims["phone"].(string)
			accountType, _ := claims["accountType"].(string)

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, UserPhoneKey, phone)
			ctx = context.WithValue(ctx, UserAccountTypeKey, accountType)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
