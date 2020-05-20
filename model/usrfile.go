package model

import "time"

type UsrFile struct {
	ID         int `gorm:"AUTO_INCREMENT;primary_key"`
	UsrId      int
	FileId     int
	FileName   string
	CreateTime time.Time
}
