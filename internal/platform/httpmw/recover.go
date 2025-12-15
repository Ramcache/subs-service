package httpmw

import (
	"net/http"

	"go.uber.org/zap"
	plog "subs-service/internal/platform/logger"
)

func Recover() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					plog.FromContext(r.Context()).Error("panic recovered",
						zap.Any("panic", v),
						zap.Stack("stack"),
					)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
