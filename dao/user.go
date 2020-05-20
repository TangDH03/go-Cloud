package dao

import (
	"cloud/model"
	"crypto/md5"
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

const (
	UsrExpireTime = 60
)

func Register(name string, password string) (bool, string) {
	db := GetDB()
	var user model.User
	db.Where("name = ?", name).First(&user)
	if user.ID != 0 {
		return false, "user exist"
	}
	uid_salt, _ := uuid.NewRandom()
	salt := uid_salt.String()[0:8]
	md5sum := fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))
	user.Name = name
	user.Password = string(md5sum[:])
	user.Salt = salt
	token, _ := uuid.NewRandom()
	db.Create(&user)
	return true, token.String()

}

func Login(name string, password string) (bool, string) {
	db := GetDB()
	var user model.User
	db.Where("name = ?", name).First(&user)
	if user.ID == 0 {
		return false, "fail"
	}

	salt := user.Salt
	md5sum := fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))
	if md5sum != user.Password {
		return false, "fail"
	}
	//fmt.Fprintln(gin.DefaultWriter, "[login request] name and password correct")
	redisClient := GetRedis()
	key := RedisUsrTokenKey(name)
	defer redisClient.Close()
	redisClient.Do("SELECT 3")

	reply, err := redis.Int(redisClient.Do("Exists", key))
	if err != nil {
		fmt.Println(err.Error())
		return false, ""
	}
	if reply != 0 {
		//fmt.Fprintln(gin.DefaultWriter, "[login request] token exist")
		redisClient.Do("Expire", key, UsrExpireTime)
		token, _ := redis.String(redisClient.Do("Get", key))
		redisClient.Do("Expire", token, UsrExpireTime)
		return true, token
	}
	token, _ := uuid.NewRandom()
	redisClient.Do("Set", key, token.String())
	redisClient.Do("Set", token.String(), name)
	redisClient.Do("Expire", key, UsrExpireTime)
	redisClient.Do("Expire", token, UsrExpireTime)
	//fmt.Fprintln(gin.DefaultWriter, "[login request] set token success")
	return true, token.String()
}
