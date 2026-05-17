package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	count int
	reset time.Time
}

func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	buckets := map[string]*bucket{}
	var mu sync.Mutex

	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()

		mu.Lock()
		b, ok := buckets[key]
		if !ok || now.After(b.reset) {
			b = &bucket{count: 0, reset: now.Add(window)}
			buckets[key] = b
		}
		b.count++
		exceeded := b.count > limit
		mu.Unlock()

		if exceeded {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
			return
		}
		c.Next()
	}
}
