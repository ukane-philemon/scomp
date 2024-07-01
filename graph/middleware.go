package graph

import (
	"context"
	"net/http"

	"github.com/ukane-philemon/scomp/internal/auth"
)

const (
	jwtHeader   = "SCOMP-Authentication-Token"
	adminCtxKey = "adminID"
)

// AuthMiddleware ensures the the correct and valid auth token is provided in
// this request.
func AuthMiddleware(authRepo auth.Repository) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			authToken := req.Header.Get(jwtHeader)
			if authToken == "" {
				next.ServeHTTP(res, req)
				return
			}

			uniqueID, validToken := authRepo.IsValid(authToken)
			if !validToken {
				http.Error(res, "not authorized", http.StatusForbidden)
				return
			}

			// Set the adminCtxKey for use by subsequent handlers.
			req = req.WithContext(context.WithValue(req.Context(), adminCtxKey, uniqueID))
			next.ServeHTTP(res, req)
		})
	}
}

// reqAuthenticated checks that the request is authenticated.
func reqAuthenticated(ctx context.Context) bool {
	adminID := ctx.Value(adminCtxKey)
	if adminID == nil {
		return false
	}
	return adminID.(string) != ""
}
