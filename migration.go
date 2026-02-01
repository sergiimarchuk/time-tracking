package main

import "log"

// UnifiedMigration - Migrate to unified minutes-based system
func UnifiedMigration() error {
	log.Println("🔄 Starting UNIFIED migration: Minutes-based time tracking")

	// Step 1: Add minutes column if it doesn't exist
	_, err := db.Exec(`
		ALTER TABLE worklogs ADD COLUMN minutes INTEGER DEFAULT 0
	`)
	if err != nil {
		log.Println("⚠️  Minutes column might already exist (this is ok):", err)
	} else {
		log.Println("✅ Minutes column added successfully")
	}

	// Step 2: Migrate ALL existing data from hours to minutes
	result, err := db.Exec(`
		UPDATE worklogs 
		SET minutes = CAST(ROUND(hours * 60) AS INTEGER)
		WHERE minutes = 0 OR minutes IS NULL
	`)

	if err != nil {
		log.Println("❌ Migration error:", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("✅ Migrated %d records to minutes-based system\n", rowsAffected)

	// Step 3: Update hours column to be calculated from minutes (for display)
	result, err = db.Exec(`
		UPDATE worklogs 
		SET hours = CAST(minutes AS REAL) / 60.0
		WHERE minutes > 0
	`)

	if err != nil {
		log.Println("⚠️  Warning updating hours column:", err)
	} else {
		log.Println("✅ Recalculated hours column from minutes")
	}

	log.Println("🎉 UNIFIED MIGRATION COMPLETE!")
	log.Println("📊 System now uses MINUTES as primary storage")

	return nil
}
