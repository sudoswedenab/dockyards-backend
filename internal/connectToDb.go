package internal

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"sync"
	"time"
)

var DB *gorm.DB

func ConnectToDB(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("trying to connect")
	dsn := os.Getenv("DB")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	DB = db
	if err != nil {
		fmt.Println("Failed to connect to database, trying again..")
		time.Sleep(time.Second * 3)
		ConnectToDB(wg)
	}
	return
}
