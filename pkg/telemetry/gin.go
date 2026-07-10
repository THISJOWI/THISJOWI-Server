package telemetry

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func GinMiddleware(serviceName string) gin.HandlerFunc {
	tp := otel.GetTracerProvider()
	return otelgin.Middleware(serviceName, otelgin.WithTracerProvider(tp))
}

func RequestIDAttribute() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if requestID, exists := c.Get("request_id"); exists {
			if rid, ok := requestID.(string); ok && rid != "" {
				span := trace.SpanFromContext(c.Request.Context())
				span.SetAttributes(attribute.String("http.request_id", rid))
			}
		}
	}
}
