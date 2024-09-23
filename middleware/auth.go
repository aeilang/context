package middleware

import (
	"context"
	"net/http"
	"strings"
)

func Auth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// "Authorization: Bearer jwttoken"
		auth := r.Header.Get("Authorization")
		ls := strings.Split(auth, " ")
		if len(ls) != 2 && ls[0] != "Bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		userId := parseToken(ls[1])
		ctx := context.WithValue(r.Context(), userKey{}, userId)
		next.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(fn)
}

type userKey struct{}

func GetUserIdFromCtx(ctx context.Context) int {
	userID, _ := ctx.Value(userKey{}).(int)
	return userID
}

func parseToken(token string) int {
	return 1
}
