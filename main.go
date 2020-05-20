package main

import (
	"cloud/handler"
	"cloud/jwt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

var router *gin.Engine
var once sync.Once

func InitRouter() {
	router = gin.Default()
	router.GET("/", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{
			"message": "OK",
		})
	})
	router.POST("/register", handler.Register)
	router.POST("/login", handler.Login)
	v1 := router.Group("/file")
	v1.Use(jwt.Auth())
	{
		v1.POST("/test", func(context *gin.Context) {
			name, _ := context.Get("usrName")
			context.JSON(http.StatusOK, gin.H{
				"message": "success",
				"name":    name,
			})
		})
		v1.POST("/upload", handler.Upload)
		v1.POST("/download", handler.Download)
	}

}
func GetRouter() *gin.Engine {
	if router == nil {
		once.Do(InitRouter)
	}
	return router
}
func main() {
	router := GetRouter()
	router.Run(":8080")
}
