package handler

import (
	"cloud/dao"
	"cloud/mr"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

const resourceDir = "./resource/"

func Upload(context *gin.Context) {
	md5sum := context.PostForm("md5sum")
	file, _ := context.FormFile("file")
	fragment := context.PostForm("fragment")
	usrName := context.Value("usrName")
	ok := false
	//context.SaveUploadedFile(file, resourceDir+file.Filename)
	if fragment == "no" {
		ok = uploadSmall(md5sum, file, usrName.(string))
	}
	if ok {
		context.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	} else {
		context.JSON(http.StatusOK, gin.H{
			"messgae": "ok",
		})
	}
}

func Download(context *gin.Context) {
	usrName := context.Value("usrName")
	fileName := context.PostForm("filename")
	location := dao.FileLocation(usrName.(string), fileName)
	context.Header("Content-Description", "File Transfer")
	context.Header("Content-Transfer-Encoding", "binary")
	context.Header("Content-Disposition", "attachment; filename="+fileName)
	context.Header("Content-Type", "application/octet-stream")
	//fmt.Fprintf(gin.DefaultWriter, fmt.Sprintf("location in %v", location))
	context.File(location)
}

func uploadSmall(md5sum string, file *multipart.FileHeader, usrName string) bool {
	exists, fileId := dao.HaveFile(md5sum)
	if exists {
		// num++
		// fmt.Println(gin.DefaultWriter, num)
		dao.LinkUsrFile(fileId, usrName, file.Filename)
		return true
	}
	master := mr.GetMaster()
	// if master == nil {
	// 	fmt.Fprintln(gin.DefaultWriter,
	// 		fmt.Sprintf("inital master is null"))
	// }
	// fmt.Fprintln(gin.DefaultWriter,
	// 	fmt.Sprintf("master's worker num is%v", master.WorkersNum()))
	// return true
	result := make(chan bool, 2)
	var job mr.UploadJob
	job.File = file
	job.Md5sum = md5sum
	job.UsrName = usrName
	job.Dir = resourceDir

	master.Send(result, job)
	re := <-result
	return re

}
