package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// Home page
func HomePage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Work Time Tracker",
	})
}

// Login page
func LoginPage(c *gin.Context) {
	timeout := c.Query("timeout")

	c.HTML(http.StatusOK, "login.html", gin.H{
		"timeout": timeout == "1",
	})
}

// Login handler
func LoginHandler(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	user, err := GetUserByUsername(username)
	if err != nil || !CheckPassword(password, user.Password) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"error": "Invalid username or password",
		})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("username", user.Username)
	session.Save()

	c.Redirect(http.StatusFound, "/dashboard")
}

// Dashboard
func DashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"username": GetCurrentUsername(c),
	})
}

// New work log form
func NewWorkLogPage(c *gin.Context) {
	c.HTML(http.StatusOK, "new_worklog.html", gin.H{})
}

// CreateWorkLogHandler — MINUTES FIRST
// CreateWorkLogHandler — SUPPORTS BOTH MODES
func CreateWorkLogHandler(c *gin.Context) {
	date := c.PostForm("date")
	description := c.PostForm("description")
	mode := c.PostForm("mode") // "minutes" or "time-range"
	userID := GetCurrentUserID(c)
	username := GetCurrentUsername(c)
	
	var minutes int
	var err error
	
	// Определяем режим ввода
	if mode == "time-range" {
		// Режим: начало/конец времени
		timeFrom := c.PostForm("time_from")
		timeTo := c.PostForm("time_to")
		
		if timeFrom == "" || timeTo == "" {
			pref, _ := LoadUserPreference(username)
			c.HTML(http.StatusOK, "new_worklog_v2.html", gin.H{
				"error":              "Please enter start and end time",
				"username":           username,
				"workingMode":        pref.WorkingMode,
				"defaultDescription": pref.DefaultDescription,
			})
			return
		}
		
		// Парсим время
		layout := "15:04"
		from, err1 := time.Parse(layout, timeFrom)
		to, err2 := time.Parse(layout, timeTo)
		
		if err1 != nil || err2 != nil {
			pref, _ := LoadUserPreference(username)
			c.HTML(http.StatusOK, "new_worklog_v2.html", gin.H{
				"error":              "Invalid time format (use HH:MM)",
				"username":           username,
				"workingMode":        pref.WorkingMode,
				"defaultDescription": pref.DefaultDescription,
			})
			return
		}
		
		// Вычисляем разницу в минутах
		duration := to.Sub(from)
		if duration <= 0 {
			pref, _ := LoadUserPreference(username)
			c.HTML(http.StatusOK, "new_worklog_v2.html", gin.H{
				"error":              "End time must be after start time",
				"username":           username,
				"workingMode":        pref.WorkingMode,
				"defaultDescription": pref.DefaultDescription,
			})
			return
		}
		
		minutes = int(duration.Minutes())
		
	} else {
		// Режим: прямой ввод минут
		minutesStr := c.PostForm("minutes")
		minutes, err = strconv.Atoi(strings.TrimSpace(minutesStr))
		
		if err != nil || minutes <= 0 {
			pref, _ := LoadUserPreference(username)
			c.HTML(http.StatusOK, "new_worklog_v2.html", gin.H{
				"error":              "Invalid minutes value",
				"username":           username,
				"workingMode":        pref.WorkingMode,
				"defaultDescription": pref.DefaultDescription,
			})
			return
		}
	}
	
	// Сохраняем в базу
	hours := float64(minutes) / 60.0
	_, err = db.Exec(
		`INSERT INTO worklogs (user_id, date, description, minutes, hours)
		 VALUES (?, ?, ?, ?, ?)`,
		userID, date, description, minutes, hours,
	)
	
	if err != nil {
		pref, _ := LoadUserPreference(username)
		c.HTML(http.StatusOK, "new_worklog_v2.html", gin.H{
			"error":              "Save error: " + err.Error(),
			"username":           username,
			"workingMode":        pref.WorkingMode,
			"defaultDescription": pref.DefaultDescription,
		})
		return
	}
	
	c.Redirect(http.StatusFound, "/worklog/new?success=1")
}



// WorkLog list
func WorkLogListPage(c *gin.Context) {
	userID := GetCurrentUserID(c)

	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	search := c.Query("search")

	query := `SELECT id, date, description, hours, COALESCE(minutes,0)
	          FROM worklogs WHERE user_id = ?`
	args := []interface{}{userID}

	if dateFrom != "" {
		query += " AND date >= ?"
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		query += " AND date <= ?"
		args = append(args, dateTo)
	}
	if search != "" {
		query += " AND description LIKE ?"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY date DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.HTML(http.StatusOK, "worklog_list.html", gin.H{
			"error": "Error loading data",
		})
		return
	}
	defer rows.Close()

	var logs []WorkLog
	totalMinutes := 0

	for rows.Next() {
		var log WorkLog
		var dateStr string
		var minutes int

		rows.Scan(&log.ID, &dateStr, &log.Description, &log.Hours, &minutes)
		log.Date, _ = time.Parse("2006-01-02", dateStr)
		log.Minutes = minutes

		totalMinutes += minutes
		logs = append(logs, log)
	}

	c.HTML(http.StatusOK, "worklog_list.html", gin.H{
		"logs":            logs,
		"totalMinutes":    totalMinutes,
		"totalHours":      float64(totalMinutes) / 60.0,
		"totalHoursInt":   totalMinutes / 60,
		"totalMinutesRem": totalMinutes % 60,
	})
}

// Edit page
func EditWorkLogPage(c *gin.Context) {
	id := c.Param("id")

	var log WorkLog
	var dateStr string
	var minutes int

	err := db.QueryRow(
		`SELECT id, date, description, hours, COALESCE(minutes,0)
		 FROM worklogs WHERE id = ?`, id,
	).Scan(&log.ID, &dateStr, &log.Description, &log.Hours, &minutes)

	if err != nil {
		c.Redirect(http.StatusFound, "/worklog/list")
		return
	}

	log.Date, _ = time.Parse("2006-01-02", dateStr)
	log.Minutes = minutes

	c.HTML(http.StatusOK, "edit_worklog.html", gin.H{"log": log})
}

// Update handler — MINUTES FIRST
func UpdateWorkLogHandler(c *gin.Context) {
	id := c.Param("id")
	date := c.PostForm("date")
	description := c.PostForm("description")
	minutesStr := c.PostForm("minutes")

	minutes, err := strconv.Atoi(strings.TrimSpace(minutesStr))
	if err != nil || minutes < 0 {
		c.HTML(http.StatusOK, "edit_worklog.html", gin.H{
			"error": "Invalid minutes value",
		})
		return
	}

	hours := float64(minutes) / 60.0

	_, err = db.Exec(
		`UPDATE worklogs
		 SET date = ?, description = ?, minutes = ?, hours = ?
		 WHERE id = ?`,
		date, description, minutes, hours, id,
	)

	if err != nil {
		c.HTML(http.StatusOK, "edit_worklog.html", gin.H{
			"error": "Update error",
		})
		return
	}

	c.Redirect(http.StatusFound, "/worklog/list")
}

// Delete
func DeleteWorkLogHandler(c *gin.Context) {
	id := c.Param("id")
	db.Exec("DELETE FROM worklogs WHERE id = ?", id)
	c.Redirect(http.StatusFound, "/worklog/list")
}

// Logout
func LogoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}

// Export to Excel
func ExportWorkLogHandler(c *gin.Context) {
	userID := GetCurrentUserID(c)
	username := GetCurrentUsername(c)

	rows, err := db.Query(
		`SELECT date, description, hours, COALESCE(minutes,0)
		 FROM worklogs WHERE user_id = ? ORDER BY date DESC`, userID,
	)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching worklogs")
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	sheet := "Work Hours"
	index, err := f.NewSheet(sheet)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating Excel sheet")
		return
	}
	f.SetActiveSheet(index)

	f.SetCellValue(sheet, "A1", "Date")
	f.SetCellValue(sheet, "B1", "Description")
	f.SetCellValue(sheet, "C1", "Minutes")
	f.SetCellValue(sheet, "D1", "Hours")

	row := 2
	totalMinutes := 0

	for rows.Next() {
		var date, desc string
		var hours float64
		var minutes int

		rows.Scan(&date, &desc, &hours, &minutes)

		t, _ := time.Parse("2006-01-02", date)
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), t.Format("02.01.2006"))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), desc)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), minutes)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), hours)

		totalMinutes += minutes
		row++
	}

	f.SetCellValue(sheet, fmt.Sprintf("B%d", row+1), "TOTAL")
	f.SetCellValue(sheet, fmt.Sprintf("C%d", row+1), totalMinutes)
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row+1), float64(totalMinutes)/60)

	file := fmt.Sprintf("worklog_%s_%s.xlsx", username, time.Now().Format("2006-01-02"))
	c.Header("Content-Disposition", "attachment; filename="+file)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	f.Write(c.Writer)
}

// Registration page
func RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{})
}

// Registration handler
func RegisterHandler(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	passwordConfirm := c.PostForm("password_confirm")

	if username == "" || password == "" {
		c.HTML(http.StatusOK, "register.html", gin.H{"error": "Please fill in all fields"})
		return
	}

	if len(username) < 3 {
		c.HTML(http.StatusOK, "register.html", gin.H{"error": "Username too short"})
		return
	}

	if len(password) < 6 {
		c.HTML(http.StatusOK, "register.html", gin.H{"error": "Password too short"})
		return
	}

	if password != passwordConfirm {
		c.HTML(http.StatusOK, "register.html", gin.H{"error": "Passwords do not match"})
		return
	}

	existing, _ := GetUserByUsername(username)
	if existing != nil {
		c.HTML(http.StatusOK, "register.html", gin.H{"error": "User already exists"})
		return
	}

	if err := CreateUser(username, password); err != nil {
		c.HTML(http.StatusOK, "register.html", gin.H{"error": err.Error()})
		return
	}

	user, _ := GetUserByUsername(username)
	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("username", user.Username)
	session.Save()

	c.Redirect(http.StatusFound, "/dashboard")
}

// Reports page
// Reports page - ENHANCED VERSION with minutes support
func ReportsPage(c *gin.Context) {
	userID := GetCurrentUserID(c)

	rows, err := db.Query(`
		SELECT date, COALESCE(minutes,0)
		FROM worklogs
		WHERE user_id = ?
		ORDER BY date ASC
	`, userID)

	if err != nil {
		c.HTML(http.StatusOK, "reports.html", gin.H{
			"error": "Error loading data",
		})
		return
	}
	defer rows.Close()

	var dates []string
	var hours []float64
	var minutes []int
	totalMinutes := 0

	// Maps for grouping
	monthsMap := make(map[string]int)  // minutes per month
	weeksMap := make(map[string]int)   // minutes per week

	for rows.Next() {
		var date string
		var mins int
		rows.Scan(&date, &mins)

		t, _ := time.Parse("2006-01-02", date)

		// Daily data
		dates = append(dates, t.Format("02.01"))
		hours = append(hours, float64(mins)/60.0)
		minutes = append(minutes, mins)
		totalMinutes += mins

		// Group by month
		monthKey := t.Format("2006-01")
		monthsMap[monthKey] += mins

		// Group by week (ISO week)
		year, week := t.ISOWeek()
		weekKey := fmt.Sprintf("%d-W%02d", year, week)
		weeksMap[weekKey] += mins
	}

	// Convert months map to slices
	var months []string
	var monthMinutes []int
	var monthHours []float64
	for month, mins := range monthsMap {
		t, _ := time.Parse("2006-01", month)
		months = append(months, t.Format("01/2006"))
		monthMinutes = append(monthMinutes, mins)
		monthHours = append(monthHours, float64(mins)/60.0)
	}

	// Convert weeks map to slices
	var weeks []string
	var weekMinutes []int
	var weekHours []float64
	for week, mins := range weeksMap {
		weeks = append(weeks, week)
		weekMinutes = append(weekMinutes, mins)
		weekHours = append(weekHours, float64(mins)/60.0)
	}

	// Calculate averages
	avgMinutes := 0
	avgHours := 0.0
	if len(dates) > 0 {
		avgMinutes = totalMinutes / len(dates)
		avgHours = float64(avgMinutes) / 60.0
	}

	// Total hours breakdown
	totalHours := totalMinutes / 60
	totalMinutesRemainder := totalMinutes % 60

	c.HTML(http.StatusOK, "reports.html", gin.H{
		// Daily data
		"dates":   dates,
		"hours":   hours,
		"minutes": minutes,

		// Monthly data
		"months":       months,
		"monthHours":   monthHours,
		"monthMinutes": monthMinutes,

		// Weekly data
		"weeks":       weeks,
		"weekHours":   weekHours,
		"weekMinutes": weekMinutes,

		// Totals
		"totalMinutes":          totalMinutes,
		"totalHours":            float64(totalMinutes) / 60.0,
		"totalHoursInt":         totalHours,
		"totalMinutesRemainder": totalMinutesRemainder,

		// Averages
		"avgMinutes": avgMinutes,
		"avgHours":   avgHours,

		// Count
		"daysCount": len(dates),
	})
}
