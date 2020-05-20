package model

import "time"

type File struct {
	ID         int `gorm:"AUTO_INCREMENT;primary_key"`
	Md5sum     string
	Location   string
	CreateTime time.Time
}
