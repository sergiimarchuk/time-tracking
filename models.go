package main

import "time"

type User struct {
	ID       int
	Username string
	Password string
}

type WorkLog struct {
	ID          int
	UserID      int
	Date        time.Time
	Description string

	Minutes int     // ✅ primary time storage
	Hours   float64 // derived / display only
}
