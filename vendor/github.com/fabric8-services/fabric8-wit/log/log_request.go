package log

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/goadesign/goa"

	"context"
)

type middlewareKey int

const (
	// ReqIDKey is the context key used by the goa RequestID middleware to store the request ID value.
	reqIDKey middlewareKey = iota + 1
)

// LogRequest creates a request logger for the goa middleware.
// This goa middleware is aware of the RequestID middleware and identity id
// if registered after it leverages the request and identity ID for logging.
// If verbose is true then the middlware logs the request and response bodies.
func LogRequest(verbose bool) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			reqID := ctx.Value(reqIDKey)
			if reqID == nil {
				reqID = shortID()
			}
			ctx = goa.WithLogContext(ctx, "req_id", reqID)

			startedAt := time.Now()
			r := goa.ContextRequest(ctx)

			reqStartedProperties := map[string]interface{}{
				r.Method: r.URL.String(),
				"from":   from(req),
				"ctrl":   goa.ContextController(ctx),
				"action": goa.ContextAction(ctx),
			}

			identityID, err := extractIdentityID(ctx)
			if err == nil {
				reqStartedProperties["identity_id"] = identityID
			}

			Info(ctx, reqStartedProperties, "request started")

			if verbose {
				if len(r.Header) > 0 {
					properties := make(map[string]interface{}, len(r.Header))
					for k, v := range r.Header {
						properties[string(k)] = v
					}
					Info(ctx, properties, "request headers")
				}
				if len(r.Params) > 0 {
					properties := make(map[string]interface{}, len(r.Params))
					for k, v := range r.Params {
						properties[string(k)] = v
					}
					Info(ctx, properties, "request params")
				}
				if r.ContentLength > 0 {
					if mp, ok := r.Payload.(map[string]interface{}); ok {
						properties := make(map[string]interface{}, len(mp))
						for k, v := range mp {
							properties[string(k)] = v
						}
						Info(ctx, properties, "request payload")
					} else {
						// Not the most efficient but this is used for debugging
						js, err := json.Marshal(r.Payload)
						if err != nil {
							js = []byte("<invalid JSON>")
						}
						Info(ctx, map[string]interface{}{"raw": string(js)}, "payload")
					}
				}
			}
			err = h(ctx, rw, req)
			resp := goa.ContextResponse(ctx)

			timeInMilli := time.Since(startedAt).Seconds() * 1e3
			reqCompletedProperties := map[string]interface{}{
				"status":        resp.Status,
				"bytes":         resp.Length,
				"duration":      timeInMilli,
				"duration_unit": "ms",
				"ctrl":          goa.ContextController(ctx),
				"action":        goa.ContextAction(ctx),
			}
			if code := resp.ErrorCode; code != "" {
				reqCompletedProperties["error"] = code
			}
			if identityID != "" {
				reqCompletedProperties["identity_id"] = identityID
			}
			Info(ctx, reqCompletedProperties, "completed")

			return err
		}
	}
}

// shortID produces a "unique" 6 bytes long string.
// Do not use as a reliable way to get unique IDs, instead use for things like logging.
func shortID() string {
	b := make([]byte, 6)
	io.ReadFull(rand.Reader, b)
	return base64.StdEncoding.EncodeToString(b)
}

// from makes a best effort to compute the request client IP.
func from(req *http.Request) string {
	if f := req.Header.Get("X-Forwarded-For"); f != "" {
		return f
	}
	f := req.RemoteAddr
	ip, _, err := net.SplitHostPort(f)
	if err != nil {
		return f
	}
	return ip
}
