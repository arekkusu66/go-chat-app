package models

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB


func InitDB() error {
	db, err := gorm.Open(sqlite.Open("sqlite.db"), &gorm.Config{})

	if err != nil {
		return err
	}

	if err = db.AutoMigrate(

		&ChatRoom{}, 
		&DM{}, 
		&Message{}, 
		&Notification{}, 
		&User{},
		&SessionData{}, 
		&AuthVerification{},
		&Setting{},
		
		); 
		err != nil {	
		return err
	}

	DB = db

	return nil
}