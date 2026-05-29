package servermw

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/avalanchian/learn-backend/ctxutil"
	"github.com/avalanchian/learn-backend/trace"
	"github.com/google/uuid"
)

func Trace(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		TraceID, err := uuid.Parse(r.Header.Get("X-Trace-Id"))
		if err != nil {
			TraceID = uuid.New()
		}
		ReqID, err := uuid.Parse(r.Header.Get("X-Request-Id"))
		if err != nil {
			ReqID = uuid.New()
		}

		trace := trace.Trace{TraceID: TraceID, RequestID: ReqID}
		ctx = ctxutil.WithValue(ctx, trace)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	}
}

func Log(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trace, ok := ctxutil.Value[trace.Trace](r.Context())

		var prefix string
		if ok {
			prefix = fmt.Sprintf("%s %s: [%s %s]: ", r.Method, r.URL, trace.TraceID, trace.RequestID)
		} else {
			prefix = fmt.Sprintf("%s %s: ", r.Method, r.URL)
		}

		logger := log.New(os.Stderr, prefix, log.LstdFlags)
		ctx := ctxutil.WithValue(r.Context(), logger)
		r = r.Clone(ctx)

		h.ServeHTTP(w, r)
	}
}

func Recovery(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				logger, ok := ctxutil.Value[*log.Logger](r.Context())
				if !ok {
					log.Printf("%s %s: panic: %v\n%s", r.Method, r.URL, err, stack)
				} else {
					logger.Printf("panic: %v\n%s", err, stack)
				}

				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("internal server error"))
			}
		}()
		h.ServeHTTP(w, r)
	}
}

func RecordResponse(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rrw := &RecordingResponseWriter{RW: w}
		start := time.Now()
		h.ServeHTTP(rrw, r)
		elapsed := time.Since(start)

		logger, ok := ctxutil.Value[*log.Logger](r.Context())
		if !ok {
			log.Printf("%s %s: %d %s: %d bytes in %s", r.Method, r.URL, rrw.StatusCode, http.StatusText(rrw.StatusCode), rrw.Bytes, elapsed)
		}
		logger.Printf("%d %s: %d bytes in %s", rrw.StatusCode, http.StatusText(rrw.StatusCode), rrw.Bytes, elapsed)
	}
}

type RecordingResponseWriter struct {
	RW         http.ResponseWriter
	StatusCode int
	Bytes      int
}

func (w *RecordingResponseWriter) WriteHeader(statusCode int) {
	if w.StatusCode == 0 {
		w.StatusCode = statusCode
	}
	w.RW.WriteHeader(statusCode)
}

func (w *RecordingResponseWriter) Header() http.Header {
	return w.RW.Header()
}

func (w *RecordingResponseWriter) Write(b []byte) (int, error) {
	if w.StatusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.RW.Write(b)
	w.Bytes += n
	return n, err
}
