package localstate

import (
	"context"

	"github.com/gin-gonic/gin"
)

// NormalizeContext extracts the stable *http.Request context from a pooled *gin.Context,
// preventing data races when database/sql background goroutines inspect Context.Done()/Err()
// after Gin returns the pooled Context to its sync.Pool.
func NormalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	if gc, ok := ctx.(*gin.Context); ok && gc.Request != nil {
		return gc.Request.Context()
	}
	return ctx
}
