package clientmw

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/avalanchian/learn-backend/middleware/ctxutil"
	"github.com/avalanchian/learn-backend/middleware/trace"

	"github.com/google/uuid"
)

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

var _ http.RoundTripper = RoundTripFunc(nil)

func logExec(name string) func() {
	log.Printf("middleware: begin %s", name)
	return func() {
		defer log.Printf("middleware: end %s", name)
	}
}

func TimeRequest(rt http.RoundTripper) RoundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		defer logExec("TimeRequest")() //logging for educational purposes

		start := time.Now()
		resp, err := RoundTrip(r) // calls the next middleware
		if err != nil {
			log.Printf("%s %s errored after %s", r.Method, r.URL, time.Since(start))
			return nil, err
		}

		log.Printf(
			"%s %s: %d %s in %s",
			r.Method,
			r.URL,
			resp.StatusCode,
			resp.StatusText(resp.StatusCode),
			time.Since(start),
		)
		return resp, nil
	}
}

func RetryOn5xx(rt http.RoundTripper, wait time.Duration, tries int) RoundTripFunc {
	// Validate args outside of closure
	if tries <= 1 {
		panic("tries must be > 1")
	}
	if wait <= 0 {
		panic("wait must be > 0")
	}

	return func(r *http.Request) (*http.Response, error) {
		defer logExec("RetryOn5xx")()

		var retryErrs error
		for retry := uint(0); retry < tries; retry++ {
			if retry > 0 {
				time.Sleep(wait << retry)
			}
			resp, err := rt.RoundTrip(r)
			if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
				retryErrs = errors.Join(retryErrs, err)
				continue
			}
			if retryErrs != nil {
				return nil, fmt.Errorf("failed after %d retries: %w", retry, retryErrs)
			}

			switch sc := resp.StatusCode; {
			case sc >= 200 && sc < 400:
				return resp, nil
			case sc >= 400 && sc < 500:
				return nil, fmt.Errorf("failed after %d retries: %w", retry, retryErrs)
			default:
				retryErrs = errors.Join(retryErrs, fmt.Errorf("try %d: %s", retry, resp.Status))
			}
		}
		return nil, fmt.Errorf("failed after %d retries: %w", tries, retryErrs)
	}
}

func Trace(rt http.RoundTripper) RoundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		defer logExec("Trace")()

		traceID, err := uuid.Parse(r.Header.Get("X-Trace-ID"))
		if err != nil {
			traceID = uuid.New()
		}

		trace := trace.Trace{TraceID: traceID, RequestID: uuid.New()}

		ctx := ctxutil.WithValue(r.Context(), trace) // retrieve with ctxutil.Value[Trace](ctx)
		r = r.WithContext(ctx)

		r.Header.Set("X-Trace-ID", trace.TraceID.String())
		r.Header.Set("X-Request-ID", trace.RequestID.String())
		return RoundTrip(r)
	}
}

func Log(rt http.RoundTripper) RoundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		defer logExec("Log")()

		trace, ok := ctxutil.Value[Trace](r.Context())
		if ok {
			prefix := fmt.Sprintf("%s %s: [%s %s]: ", r.Method, r.URL, trace.TraceID, trace.RequestID)
		} else {
			prefix := fmt.Sprintf("%s %s: ", r.Method, r.URL)
		}

		logger := log.New(os.Stderr, prefix, log.LstdFlags|log.Lshortfile)
		ctx := ctxutil.WithValue(r.Context(), logger)
		r = r.WithContext(ctx)

		start := time.Now()
		resp, err := rt.RoundTrip(r)
		if err != nil {
			logger.Printf("errored after %s: %s", time.Since(start), err)
			return nil, err
		}

		logger.Printf("%d %s in %s", resp.StatusCode, resp.StatusText(resp.StatusCode), time.Since(start))
		return resp, nil
	}
}
