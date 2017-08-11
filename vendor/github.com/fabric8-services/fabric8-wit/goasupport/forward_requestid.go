package goasupport

import (
	"context"

	"github.com/goadesign/goa/client"
	"github.com/goadesign/goa/middleware"
)

func ForwardContextRequestID(ctx context.Context) context.Context {
	reqID := middleware.ContextRequestID(ctx)
	if reqID != "" {
		return client.SetContextRequestID(ctx, reqID)
	}
	return ctx
}
