package main

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
)

var jwtSecret = []byte("your-super-secret-jwt-key-change-in-production")

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Генерация JWT токена
func GenerateJWT(userID int, username string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Проверка JWT токена
func ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, err
}

// Middleware для проверки JWT
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		claims, err := ValidateJWT(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// API: Логин
func APILogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, err := GetUserByUsername(req.Username)
	if err != nil || !CheckPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := GenerateJWT(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// API: Регистрация
func APIRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	existingUser, _ := GetUserByUsername(req.Username)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	err := CreateUser(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	user, _ := GetUserByUsername(req.Username)
	token, _ := GenerateJWT(user.ID, user.Username)

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// API: Получить все записи
func APIGetWorkLogs(c *gin.Context) {
	userID := c.GetInt("user_id")

	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	search := c.Query("search")

	query := `SELECT id, date, description, hours FROM worklogs WHERE user_id = ?`
	args := []interface{}{userID}

	if dateFrom != "" {
		query += ` AND date >= ?`
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		query += ` AND date <= ?`
		args = append(args, dateTo)
	}
	if search != "" {
		query += ` AND description LIKE ?`
		args = append(args, "%"+search+"%")
	}

	query += ` ORDER BY date DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id int
		var date, description string
		var hours float64
		rows.Scan(&id, &date, &description, &hours)

		logs = append(logs, map[string]interface{}{
			"id":          id,
			"date":        date,
			"description": description,
			"hours":       hours,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": logs})
}

// API: Создать запись
func APICreateWorkLog(c *gin.Context) {
	userID := c.GetInt("user_id")

	var req struct {
		Date        string  `json:"date" binding:"required"`
		Description string  `json:"description" binding:"required"`
		Hours       float64 `json:"hours" binding:"required,min=0,max=24"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result, err := db.Exec(
		"INSERT INTO worklogs (user_id, date, description, hours) VALUES (?, ?, ?, ?)",
		userID, req.Date, req.Description, req.Hours,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create worklog"})
		return
	}

	id, _ := result.LastInsertId()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Worklog created",
		"id":      id,
	})
}

// API: Обновить запись
func APIUpdateWorkLog(c *gin.Context) {
	userID := c.GetInt("user_id")
	id := c.Param("id")

	var req struct {
		Date        string  `json:"date" binding:"required"`
		Description string  `json:"description" binding:"required"`
		Hours       float64 `json:"hours" binding:"required,min=0,max=24"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result, err := db.Exec(
		"UPDATE worklogs SET date = ?, description = ?, hours = ? WHERE id = ? AND user_id = ?",
		req.Date, req.Description, req.Hours, id, userID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update worklog"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worklog not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worklog updated"})
}

// API: Удалить запись
func APIDeleteWorkLog(c *gin.Context) {
	userID := c.GetInt("user_id")
	id := c.Param("id")

	result, err := db.Exec("DELETE FROM worklogs WHERE id = ? AND user_id = ?", id, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete worklog"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worklog not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worklog deleted"})
}

// API: Получить статистику
func APIGetStats(c *gin.Context) {
	userID := c.GetInt("user_id")

	rows, err := db.Query("SELECT date, hours FROM worklogs WHERE user_id = ? ORDER BY date ASC", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var totalHours float64
	var daysCount int

	for rows.Next() {
		var date string
		var hours float64
		rows.Scan(&date, &hours)
		totalHours += hours
		daysCount++
	}

	avgHours := 0.0
	if daysCount > 0 {
		avgHours = totalHours / float64(daysCount)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_hours": totalHours,
		"days_count":  daysCount,
		"avg_hours":   avgHours,
	})
}
