package httpmw

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	plog "subs-service/internal/platform/logger"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func AccessLog() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)

			// если handler вообще ничего не написал
			if sw.status == 0 {
				sw.status = http.StatusOK
			}

			log := plog.FromContext(r.Context()).With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", sw.status),
				zap.Int("bytes", sw.bytes),
				zap.Duration("duration", time.Since(start)),
				zap.String("remote_addr", r.RemoteAddr),
			)

			switch {
			case sw.status >= 500:
				log.Error("http request completed")
			case sw.status >= 400:
				log.Warn("http request completed")
			default:
				log.Info("http request completed")
			}
		})
	}
}
