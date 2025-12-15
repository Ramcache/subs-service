package httpmw

import (
	"net/http"

	"go.uber.org/zap"
	"subs-service/internal/platform/logger"
)

func ContextLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := logger.WithLogger(r.Context(), log)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
