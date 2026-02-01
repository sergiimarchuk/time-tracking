package main

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

// Хешируем пароль
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// Проверяем пароль
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Создаём пользователя
func CreateUser(username, password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, hash)
	return err
}

// Получаем пользователя по username
func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := db.QueryRow("SELECT id, username, password FROM users WHERE username = ?", username).
		Scan(&user.ID, &user.Username, &user.Password)

	if err != nil {
		return nil, err
	}
	return user, nil
}

// Middleware для проверки авторизации
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")

		if userID == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Next()
	}
}

// Получить ID текущего пользователя из сессии
func GetCurrentUserID(c *gin.Context) int {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID == nil {
		return 0
	}

	return userID.(int)
}

// Получить имя текущего пользователя
func GetCurrentUsername(c *gin.Context) string {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		return ""
	}

	return username.(string)
}
