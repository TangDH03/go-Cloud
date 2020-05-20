package dao

import (
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var db *gorm.DB
var once sync.Once
var pool *redis.Pool
var redisOnce sync.Once

func GetDB() *gorm.DB {
	once.Do(initialDB)
	return db
}

func initialDB() {
	if db == nil {
		var err error
		db, err = gorm.Open("mysql", "root:qazwsx@34@/cloud?charset=utf8&parseTime=True&loc=Local")
		db.DB().SetMaxOpenConns(10)
		if err != nil {
			panic(err.Error())
		}
	}

}

func GetRedis() redis.Conn {
	redisOnce.Do(initialRedisPool)
	return pool.Get()
}

func initialRedisPool() {
	if pool == nil {
		pool = &redis.Pool{MaxIdle: 10,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", ":6379")
			}}
	}
}
