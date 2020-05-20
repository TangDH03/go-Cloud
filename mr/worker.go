package mr

import (
	"cloud/dao"
	"cloud/model"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

type Worker struct {
	State int
	Jobs  chan interface{}
	Reply chan interface{}
}

func (worker *Worker) Run() {
	for {
		select {
		case job := <-worker.Jobs:
			switch j := job.(type) {
			case UploadJob:
				worker.Reply <- storeFile(j)
			}
		}
	}
}

func storeFile(job UploadJob) bool {
	redisClient := dao.GetRedis()
	defer redisClient.Close()
	//get lock
	for {
		ok, _ := redis.Int(redisClient.Do("SETNX", "storeFile", "1"))
		if ok == 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	//save file
	db := dao.GetDB()
	var resourceFileName strings.Builder
	var lastInsertFile model.File
	db.Last(&lastInsertFile)
	resourceFileName.WriteString(strconv.Itoa(int(lastInsertFile.ID) + 1))
	uploadFileName := job.File.Filename
	extension := strings.Split(uploadFileName, ".")[1]
	resourceFileName.WriteString("." + extension)
	src, _ := job.File.Open()
	defer src.Close()
	out, _ := os.Create(job.Dir + resourceFileName.String())
	defer out.Close()
	_, err := io.Copy(out, src)
	if err != nil {
		fmt.Fprintln(gin.DefaultWriter,
			fmt.Sprintf("worker get a job ,complete fail"))
		return false
	}
	//insert into mysql
	var insertFile model.File
	insertFile.Md5sum = job.Md5sum
	insertFile.Location = job.Dir + resourceFileName.String()
	insertFile.CreateTime = time.Now()
	db.Create(&insertFile)
	//release lock
	defer redisClient.Do("DEL", "storeFile")
	//link usr file
	dao.LinkUsrFile(int(insertFile.ID), job.UsrName, job.File.Filename)
	//redis set cache
	redisClient.Do("Set", job.Md5sum, insertFile.ID)
	redisClient.Do("Expire", job.Md5sum, dao.FileExpireTime)
	fmt.Fprintln(gin.DefaultWriter,
		fmt.Sprintf("worker get a job ,complete success"))
	return true
}
