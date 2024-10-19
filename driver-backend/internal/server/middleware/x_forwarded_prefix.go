package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"net/url"
	"strings"
)

func getCanonicalPath(r *http.Request) string {
	// Parse the request URL
	u, err := url.Parse(r.URL.String())
	if err != nil {
		return "" // Handle error appropriately
	}

	// Normalize the path
	u.Path = u.EscapedPath()

	// Remove any trailing slashes
	if u.Path != "/" && u.Path[len(u.Path)-1] == '/' {
		u.Path = u.Path[:len(u.Path)-1]
	}

	return u.Path
}

// XForwardedPrefixMiddleware checks for the X-Forwarded-Prefix header and updates the request URL path accordingly.
func XForwardedPrefixMiddleware(atom *zap.AtomicLevel) gin.HandlerFunc {
	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zapConfig), zapcore.AddSync(colorable.NewColorableStdout()), atom)
	logger := zap.New(core, zap.Development())

	return func(c *gin.Context) {
		// Check for the X-Forwarded-Prefix header
		if forwardedPrefix := c.GetHeader("X-Forwarded-Prefix"); forwardedPrefix != "" {
			logger.Debug("Discovered X-Forwarded-Prefix headers.", zap.String("X-Forwarded-Prefix", forwardedPrefix))

			// Ensure prefix starts with a slash
			if !strings.HasPrefix(forwardedPrefix, "/") {
				forwardedPrefix = "/" + forwardedPrefix
			}

			originalPath := c.Request.URL.Path

			// Prepend the forwarded prefix to the request URL path
			c.Request.URL.Path = forwardedPrefix + c.Request.URL.Path

			// Ensure the path is cleaned up (removes any double slashes, etc.)
			c.Request.URL.Path = getCanonicalPath(c.Request)

			logger.Debug("Updated request URL path according to X-Forwarded-Prefix headers.",
				zap.String("X-Forwarded-Prefix", forwardedPrefix),
				zap.String("original_path", originalPath),
				zap.String("updated_path", c.Request.URL.Path))
		}

		// Continue processing the request
		c.Next()
	}
}
