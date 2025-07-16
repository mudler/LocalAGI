package db

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	models "github.com/mudler/LocalAGI/dbmodels"
)

var DB *gorm.DB

func ConnectDB() {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	name := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, pass, host, name)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
			NoLowerCase:   true, // preserve camelCase column names
		},
	})
	if err != nil {
		log.Fatal("Failed to connect to MySQL:", err)
	}

	if err := DB.AutoMigrate(&models.User{}, &models.Agent{}, &models.AgentMessage{}, &models.LLMUsage{}, &models.Character{}, &models.AgentState{}, &models.ActionExecution{}, &models.Reminder{}); err != nil {
		log.Fatal("Migration failed:", err)
	}
}
