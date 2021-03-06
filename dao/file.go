package dao

import (
	"cloud/model"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

const (
	FileExpireTime = 60 * 60 * 24
)

func HaveFileByName(usrName string, filename string) (bool, int) {
	var file model.UsrFile
	var usr model.User
	db := GetDB()
	db.Where("name = ? ", usrName).First(&usr)
	db.Where("file_name = ? and usr_id = ? ", filename, usr.ID).First(&file)
	if file.ID == 0 {
		return false, 0
	}
	return true, file.ID

}

func HaveFile(md5sum string) (bool, int) {
	//redis Has it
	redisClient := GetRedis()
	db := GetDB()
	defer redisClient.Close()
	exists, _ := redis.Int(redisClient.Do("Exists", md5sum))
	redisClient.Do("Expire", md5sum, FileExpireTime)
	if exists == 1 {
		fileId, _ := redis.Int(redisClient.Do("Get", md5sum))
		return true, fileId
	}
	var file model.File
	db.Where("md5sum = ?", md5sum).First(&file)
	if file.ID == 0 {
		return false, 0
	}
	redisClient.Do("Set", md5sum, file.ID)
	redisClient.Do("Expire", md5sum, FileExpireTime)
	return true, int(file.ID)

}

func HaveTmpFile(md5sum string) (bool, string) {
	redisClient := GetRedis()
	db := GetDB()
	defer redisClient.Close()
	exists, _ := redis.Int(redisClient.Do("Exists", md5sum))
	redisClient.Do("Expire", md5sum, FileExpireTime)
	if exists == 1 {
		location, _ := redis.String(redisClient.Do("Get", md5sum))
		return true, location
	}
	var file model.TmpFile
	db.Where("md5sum = ?", md5sum).First(&file)
	if file.ID == 0 {
		return false, ""
	}
	redisClient.Do("Set", md5sum, file.Location)
	redisClient.Do("Expire", md5sum, FileExpireTime)
	return true, file.Location

}

func LinkUsrFile(fileId int, usrName string, filename string) {
	var usr model.User
	var usrfile model.UsrFile
	db := GetDB()
	db.Where("name = ?", usrName).First(&usr)
	usrfile.FileId = int(fileId)
	usrfile.UsrId = int(usr.ID)
	usrfile.FileName = filename
	usrfile.CreateTime = time.Now()
	db.Create(&usrfile)
}
func StoreFile(usrName string, md5sum string,
	file *multipart.FileHeader, dir string) {
	redisClient := GetRedis()
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
	var resourceFileName strings.Builder
	var lastInsertFile model.File
	db := GetDB()
	db.Last(&lastInsertFile)
	resourceFileName.WriteString(strconv.Itoa(int(lastInsertFile.ID) + 1))
	uploadFileName := file.Filename
	extension := strings.Split(uploadFileName, ".")[1]
	resourceFileName.WriteString("." + extension)
	src, _ := file.Open()
	defer src.Close()
	out, _ := os.Create(dir + resourceFileName.String())
	defer out.Close()
	io.Copy(out, src)

	//insert into mysql

	var insertFile model.File
	insertFile.Md5sum = md5sum
	insertFile.Location = dir + resourceFileName.String()
	insertFile.CreateTime = time.Now()
	db.Create(&insertFile)
	//release lock
	redisClient.Do("DEL", "storeFile")
	//link usr file
	LinkUsrFile(int(insertFile.ID), usrName, file.Filename)
	//redis set cache
	redisClient.Do("Set", md5sum, insertFile.ID)
	redisClient.Do("Expire", md5sum, FileExpireTime)

}

func FileLocation(usrName string, fileName string) string {
	db := GetDB()
	var usr model.User
	var usrFile model.UsrFile
	db.Where("name = ? ", usrName).First(&usr)
	db.Where("usr_id = ? AND  file_name =  ? ", usr.ID, fileName).First(&usrFile)

	var file model.File
	db.Where("id = ?", usrFile.FileId).First(&file)
	// fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("userId: %v fileId: %v fileName: %v location %v",
	// 	usr.ID, usrFile.FileId, fileName, file.Location))
	return file.Location

}

func FastUpload(md5sum string, fileName string, usrName string) bool {
	exists, fileId := HaveFile(md5sum)
	if exists {
		LinkUsrFile(fileId, usrName, fileName)
		return true
	}
	panic(fmt.Sprintf("[fast Upload] file should be saved and only link it with usr,but file not be saved"))

}

func FastTmpUpload(md5sum string, fileName string, usrName string, fragment int) bool {
	exists, location := HaveTmpFile(md5sum)
	redisClient := GetRedis()
	if exists {
		for {
			len, _ := redis.Int(redisClient.Do("LLEN", usrName+fileName))
			if len == fragment {
				redisClient.Do("LPUSH", usrName+fileName, filepath.Base(location))
				redisClient.Do("Expire", usrName+fileName)
				redisClient.Close()
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		return true

	}
	panic(fmt.Sprintf("[fast Tmp Upload] file should be saved and only link it with usr,but file not be saved"))
}
