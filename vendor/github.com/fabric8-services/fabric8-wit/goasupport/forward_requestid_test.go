package goasupport

import (
	"context"
	"net/http"
	"testing"

	"net/http/httptest"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/uuid"
	"github.com/stretchr/testify/assert"
)

func TestForwardRequest(t *testing.T) {

	reqID := uuid.NewV4().String()
	ctx := context.Background()

	service := goa.New("test")
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/goo", nil)
	req.Header.Set(middleware.RequestIDHeader, reqID)

	var newCtx context.Context
	h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		newCtx = ctx
		return service.Send(ctx, 200, "ok")
	}
	rg := middleware.RequestID()(h)
	rg(ctx, rw, req)

	assert.Equal(t, middleware.ContextRequestID(newCtx), reqID)

	clientCtx := ForwardContextRequestID(newCtx)
	assert.Equal(t, client.ContextRequestID(clientCtx), reqID)
}
