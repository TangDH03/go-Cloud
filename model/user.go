package model

type User struct {
	ID       int `gorm:"AUTO_INCREMENT;unique;primary_key"`
	Name     string
	Password string
	Salt     string
}
