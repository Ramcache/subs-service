package httpmw

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	plog "subs-service/internal/platform/logger"
)

func RequestLogger(base *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			l := base
			if reqID != "" {
				l = base.With(zap.String("request_id", reqID))
			}
			ctx := plog.WithLogger(r.Context(), l)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
