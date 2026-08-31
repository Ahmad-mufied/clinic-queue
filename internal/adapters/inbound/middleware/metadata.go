package middleware

import (
	"clinic-queue/internal/core/domain"

	"github.com/labstack/echo/v4"
)

// ClientMetadataMiddleware extracts client forensic metadata (IP, User-Agent, Request-ID)
// from the HTTP request and injects it into the request's context.Context.
func ClientMetadataMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqID := c.Response().Header().Get(echo.HeaderXRequestID)
			if reqID == "" {
				reqID = c.Request().Header.Get(echo.HeaderXRequestID)
			}

			meta := domain.ClientMetadata{
				ClientIP:  c.RealIP(),
				UserAgent: c.Request().UserAgent(),
				RequestID: reqID,
			}

			ctx := domain.WithClientMetadata(c.Request().Context(), meta)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
