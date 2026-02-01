package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

/*
	=========================================================
	  USER PREFERENCES

=========================================================
*/
const PREFERENCES_DIR = "./state-web-form"

// Preference structure
type UserPreference struct {
	Username           string
	WorkingMode        string // "time-range" or "minutes"
	DefaultDescription string
}

// Initialize preferences directory
func InitPreferencesDir() error {
	return os.MkdirAll(PREFERENCES_DIR, 0755)
}

// Get preference file path
func getUserPreferenceFile(username string) string {
	return filepath.Join(PREFERENCES_DIR, username+".config")
}

// Load preferences
func LoadUserPreference(username string) (*UserPreference, error) {
	filePath := getUserPreferenceFile(username)
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &UserPreference{
				Username:           username,
				WorkingMode:        "time-range",
				DefaultDescription: "",
			}, nil
		}
		return nil, err
	}
	defer file.Close()
	pref := &UserPreference{Username: username}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "working-mode":
			pref.WorkingMode = value
		case "default-description":
			pref.DefaultDescription = value
		}
	}
	return pref, scanner.Err()
}

// Save preferences
func SaveUserPreference(pref *UserPreference) error {
	if err := InitPreferencesDir(); err != nil {
		return err
	}
	file, err := os.Create(getUserPreferenceFile(pref.Username))
	if err != nil {
		return err
	}
	defer file.Close()
	w := bufio.NewWriter(file)
	fmt.Fprintln(w, "# User Preferences for Work Time Tracker")
	fmt.Fprintf(w, "# Username: %s\n", pref.Username)
	fmt.Fprintln(w, "# Available modes: time-range, minutes")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "working-mode: %s\n", pref.WorkingMode)
	fmt.Fprintf(w, "default-description: %s\n", pref.DefaultDescription)
	return w.Flush()
}

/* =========================================================
   HANDLERS – PREFERENCES
========================================================= */
// Replaces NewWorkLogPage
func NewWorkLogPageWithPreferences(c *gin.Context) {
	username := GetCurrentUsername(c)
	pref, err := LoadUserPreference(username)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "new_worklog.html", gin.H{
			"error": "Failed to load preferences",
		})
		return
	}

	// Check for success parameter from redirect
	successParam := c.Query("success")
	var successMsg string
	if successParam == "1" {
		successMsg = "✅ Entry saved successfully!"
	}

	c.HTML(http.StatusOK, "new_worklog_v2.html", gin.H{
		"username":           username,
		"workingMode":        pref.WorkingMode,
		"defaultDescription": pref.DefaultDescription,
		"success":            successMsg,
	})
}

// Save preferences
func SaveUserPreferenceHandler(c *gin.Context) {
	username := GetCurrentUsername(c)
	mode := c.PostForm("mode")
	description := c.PostForm("description")
	if mode != "time-range" && mode != "minutes" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid working mode",
		})
		return
	}
	pref := &UserPreference{
		Username:           username,
		WorkingMode:        mode,
		DefaultDescription: description,
	}
	if err := SaveUserPreference(pref); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save preferences",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Get preferences
func GetUserPreferenceHandler(c *gin.Context) {
	username := GetCurrentUsername(c)
	pref, err := LoadUserPreference(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to load preferences",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"workingMode":        pref.WorkingMode,
		"defaultDescription": pref.DefaultDescription,
	})
}

/*
	=========================================================
	  MAIN

=========================================================
*/
func main() {
	// Init DB
	if err := InitDB(); err != nil {
		log.Fatal("Database error:", err)
	}

	// 🆕 UNIFIED MIGRATION - Run migration to add minutes column
	if err := UnifiedMigration(); err != nil {
		log.Println("⚠️  Migration warning:", err)
	}

	// Init preference directory
	if err := InitPreferencesDir(); err != nil {
		log.Fatal("Preferences error:", err)
	}

	r := gin.Default()

	// Sessions
	store := cookie.NewStore([]byte("super-secret-key-change-me-in-production"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("mysession", store))

	// Template helpers
	r.SetFuncMap(template.FuncMap{
		"add": func(a, b float64) float64 {
			return a + b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
	})

	// Templates & static
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	/* ===== PUBLIC ROUTES ===== */
	r.GET("/", HomePage)
	r.GET("/login", LoginPage)
	r.POST("/login", LoginHandler)
	//	r.GET("/register", RegisterPage)
	//	r.POST("/register", RegisterHandler)
	r.GET("/register", func(c *gin.Context) {
	c.AbortWithStatus(404)
	})
	r.POST("/register", func(c *gin.Context) {
    	c.AbortWithStatus(404)
	})


	/* ===== AUTH ROUTES ===== */
	authorized := r.Group("/")
	authorized.Use(AuthRequired(), CheckInactivity())
	{
		authorized.GET("/dashboard", DashboardPage)

		// 🔥 REPLACED
		authorized.GET("/worklog/new", NewWorkLogPageWithPreferences)
		authorized.POST("/worklog/create", CreateWorkLogHandler)
		authorized.GET("/worklog/list", WorkLogListPage)
		authorized.GET("/reports", ReportsPage)
		authorized.GET("/worklog/edit/:id", EditWorkLogPage)
		authorized.POST("/worklog/update/:id", UpdateWorkLogHandler)
		authorized.POST("/worklog/delete/:id", DeleteWorkLogHandler)
		authorized.GET("/worklog/export", ExportWorkLogHandler)

		// Preferences
		authorized.POST("/preferences/save", SaveUserPreferenceHandler)
		authorized.GET("/preferences/get", GetUserPreferenceHandler)

		authorized.GET("/logout", LogoutHandler)
	}

	/* ===== API ROUTES ===== */
	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", APILogin)
		api.POST("/auth/register", APIRegister)

		apiAuth := api.Group("/")
		apiAuth.Use(JWTAuthMiddleware())
		{
			apiAuth.GET("/worklogs", APIGetWorkLogs)
			apiAuth.POST("/worklogs", APICreateWorkLog)
			apiAuth.PUT("/worklogs/:id", APIUpdateWorkLog)
			apiAuth.DELETE("/worklogs/:id", APIDeleteWorkLog)
			apiAuth.GET("/stats", APIGetStats)
		}
	}

	log.Println("🚀 Server started at http://localhost:8080")
	log.Println("📡 API available at http://localhost:8080/api/v1")
	r.Run(":8080")
}
