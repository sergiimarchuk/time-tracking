package main

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"time"
)

const inactivityTimeout = 30 * time.Minute // 30 минут

// Middleware для проверки неактивности
func CheckInactivity() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		lastActivity := session.Get("last_activity")

		if lastActivity != nil {
			lastTime, ok := lastActivity.(int64)
			if ok {
				elapsed := time.Since(time.Unix(lastTime, 0))

				// Если прошло больше 30 минут - выходим
				if elapsed > inactivityTimeout {
					session.Clear()
					session.Save()
					c.Redirect(302, "/login?timeout=1")
					c.Abort()
					return
				}
			}
		}

		// Обновляем время последней активности
		session.Set("last_activity", time.Now().Unix())
		session.Save()

		c.Next()
	}
}
