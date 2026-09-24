package middleware

import (
	"context"
	"net/http"

	"loanapp/internal/auth"
)

type contextKey string

const userContextKey contextKey = "authUser"

type AuthUser struct {
	UserID string
	Role   string
}

func contextWithUser(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(userContextKey).(AuthUser)
	return u, ok
}

func AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := auth.VerifyToken(cookie.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := contextWithUser(r.Context(), AuthUser{UserID: claims.UserID, Role: claims.Role})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || user.Role != role {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
