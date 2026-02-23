package database

import (
    "fmt"
    "log"
    "os"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
    // Example: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        getEnv("DB_USER", "root"),
        getEnv("DB_PASSWORD", ""),
        getEnv("DB_HOST", "127.0.0.1"),
        getEnv("DB_PORT", "3306"),
        getEnv("DB_NAME", "ticketing_sys"),
    )

    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("❌ Failed to connect to database: ", err)
    }

    log.Println("✅ Database connected successfully")
    DB = db
}

// Helper function to read env vars with fallback
func getEnv(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}
