package jwt

import (
	"cloud/dao"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

func Auth() gin.HandlerFunc {
	return auth
}

func auth(context *gin.Context) {
	token := context.PostForm("token")
	//fmt.Fprintln(gin.DefaultWriter, token)
	if token == "" {
		context.JSON(http.StatusOK, gin.H{
			"message": "UnLogin",
		})
		context.Abort()
		return
	}
	//fmt.Fprintln(gin.DefaultWriter, "has token")
	redisClient := dao.GetRedis()
	defer redisClient.Close()
	exists, _ := redis.Int(redisClient.Do("Exists", token))
	if exists == 0 {
		context.JSON(http.StatusOK, gin.H{
			"message": "token expire",
		})
		context.Abort()
		return
	}
	name, _ := redis.String(redisClient.Do("Get", token))
	redisClient.Do("Expire", token, dao.UsrExpireTime)
	redisClient.Do("Expire", dao.RedisUsrTokenKey(name), dao.UsrExpireTime)
	context.Set("usrName", name)

}
