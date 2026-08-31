package middleware

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	echoMW "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

// NewAuthRateLimiter creates a Token Bucket rate limiter for authentication endpoints.
// Configuration: 10 requests per minute with a burst capacity of 5 per client IP.
func NewAuthRateLimiter(enabled bool) echo.MiddlewareFunc {
	if !enabled {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(ctx echo.Context) error {
				return next(ctx)
			}
		}
	}

	store := echoMW.NewRateLimiterMemoryStoreWithConfig(echoMW.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(10.0 / 60.0), // 10 req / minute
		Burst:     5,
		ExpiresIn: 3 * time.Minute,
	})

	return echoMW.RateLimiterWithConfig(echoMW.RateLimiterConfig{
		Store: store,
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			ip := ctx.RealIP()
			return ip, nil
		},
		DenyHandler: func(ctx echo.Context, identifier string, err error) error {
			return ctx.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "Too many requests. Please try again later.",
			})
		},
	})
}

// NewQueueRateLimiter creates a Token Bucket rate limiter for queue joining endpoints.
// Configuration: 30 requests per minute with a burst capacity of 10 per client IP.
func NewQueueRateLimiter(enabled bool) echo.MiddlewareFunc {
	if !enabled {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(ctx echo.Context) error {
				return next(ctx)
			}
		}
	}

	store := echoMW.NewRateLimiterMemoryStoreWithConfig(echoMW.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(30.0 / 60.0), // 30 req / minute
		Burst:     10,
		ExpiresIn: 3 * time.Minute,
	})

	return echoMW.RateLimiterWithConfig(echoMW.RateLimiterConfig{
		Store: store,
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			ip := ctx.RealIP()
			return ip, nil
		},
		DenyHandler: func(ctx echo.Context, identifier string, err error) error {
			return ctx.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "Too many requests. Please try again later.",
			})
		},
	})
}
