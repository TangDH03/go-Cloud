package handler

import (
	"cloud/dao"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Register(context *gin.Context) {
	name := context.PostForm("name")
	passwd := context.PostForm("password")
	ok, message := dao.Register(name, passwd)
	if ok {
		context.JSON(http.StatusOK, gin.H{
			"message": "success",
			"token":   message,
		})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"message": message,
		"token":   "",
	})

}

func Login(context *gin.Context) {
	name := context.PostForm("name")
	passwd := context.PostForm("password")
	ok, message := dao.Login(name, passwd)
	//fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("[login request] ok:%v messgae:%v", ok, message))
	//fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("[login request]: name:%v password:%v", name, passwd))
	if ok {
		context.JSON(http.StatusOK, gin.H{
			"message": "success",
			"token":   message,
		})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"message": "fail",
		"token":   "",
	})
}
