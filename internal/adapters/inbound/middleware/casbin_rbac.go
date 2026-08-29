package middleware

import (
	"fmt"
	"net/http"

	"github.com/casbin/casbin/v2"
	"github.com/labstack/echo/v4"
)

// NewCasbinEnforcer loads the Casbin model and policy from file paths and returns an initialized enforcer.
func NewCasbinEnforcer(modelPath, policyPath string) (*casbin.Enforcer, error) {
	enforcer, err := casbin.NewEnforcer(modelPath, policyPath)
	if err != nil {
		return nil, fmt.Errorf("initialize casbin enforcer: %w", err)
	}
	return enforcer, nil
}

// CasbinRBAC returns an Echo middleware that checks if the request is permitted by the Casbin policy.
func CasbinRBAC(enforcer *casbin.Enforcer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			method := c.Request().Method

			sub, ok := GetUserRole(c)
			if !ok || sub == "" {
				sub = "public"
			}

			allowed, err := enforcer.Enforce(sub, path, method)
			if err != nil || !allowed {
				switch sub {
				case "public":
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "Unauthorized access",
					})
				default:
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "Access denied: insufficient role privileges",
					})
				}
			}

			return next(c)
		}
	}
}
