package httpmw

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func ResponseRequestID(headerName string) func(http.Handler) http.Handler {
	if headerName == "" {
		headerName = "X-Request-ID"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rid := middleware.GetReqID(r.Context()); rid != "" {
				w.Header().Set(headerName, rid)
			}
			next.ServeHTTP(w, r)
		})
	}
}
