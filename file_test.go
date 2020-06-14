package main

import (
	"bufio"
	"bytes"
	"cloud/dao"
	"cloud/model"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

const testfiledir = "./testfile/upload/"
const testBigfiledir = "./testfile/BigUpload/"
const filechunk = 1024 * 1024 * 10

func TestUpload(t *testing.T) {
	files, _ := ioutil.ReadDir("./testfile/upload")
	router := GetRouter()
	file, err := os.Open("./testfile/notRegister.txt")
	assert.Equal(t, nil, err)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	result := make(chan bool, 10)
	var number int
	for scanner.Scan() {
		number += 1
		usr_password := scanner.Text()
		u_p := strings.Split(usr_password, " ")
		go login(router, u_p[0], u_p[1], result)
	}
	for i := 0; i < number; i++ {
		value := <-result
		assert.Equal(t, true, value)
	}
	var wg sync.WaitGroup
	Loginmap.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go uploadSmall(router, key.(string), &wg, files)
		return true
	})
	fmt.Fprintln(gin.DefaultWriter, "not finish")
	wg.Wait()
	//assert.Equal(t, false, true)
}
func TestBigUpload(t *testing.T) {
	files, _ := ioutil.ReadDir("./testfile/BigUpload")
	router := GetRouter()
	file, err := os.Open("./testfile/notRegister.txt")
	assert.Equal(t, nil, err)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	result := make(chan bool, 10)
	var number int
	for scanner.Scan() {
		number += 1

		usr_password := scanner.Text()
		u_p := strings.Split(usr_password, " ")
		go login(router, u_p[0], u_p[1], result)
		if number >= 105 {
			break
		}
	}
	for i := 0; i < number; i++ {
		value := <-result
		assert.Equal(t, true, value)
	}
	number = 0
	var wg sync.WaitGroup
	Loginmap.Range(func(key, value interface{}) bool {
		wg.Add(1)
		number++
		go uploadBig(router, key.(string), &wg, files, result)
		return true
	})
	fmt.Fprintln(gin.DefaultWriter, "not finish")
	wg.Wait()

	for i := 0; i < number; i++ {
		assert.Equal(t, true, <-result)
	}

}
func uploadBig(router *gin.Engine, token string, wg *sync.WaitGroup, files []os.FileInfo, result chan bool) {
	defer wg.Done()
	w := httptest.NewRecorder()
	re := true
	//Open file
	for i := 0; i < 3; i++ {
		file, _ := os.Open(testBigfiledir + files[rand.Intn(len(files))].Name())
		defer file.Close()
		info, _ := file.Stat()
		filesize := info.Size()
		blocks := uint64(math.Ceil(float64(filesize) / float64(filechunk)))
		allhash := md5.New()
		for j := uint64(0); j < blocks; j++ {
			hash := md5.New()
			blocksize := int(math.Min(filechunk, float64(filesize-int64(j*filechunk))))
			buf := make([]byte, blocksize)
			file.Read(buf)
			io.WriteString(hash, string(buf))
			io.WriteString(allhash, string(buf))
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			//add file,token,md5sum
			part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
			io.Copy(part, bytes.NewReader(buf))
			writer.WriteField("token", token)
			writer.WriteField("md5sum", fmt.Sprintf("%x", (hash.Sum(nil)[:])))
			writer.WriteField("fragment", strconv.Itoa(int(j)))
			writer.WriteField("blocks", strconv.Itoa(int(blocks)))
			writer.Close()

			req := httptest.NewRequest("POST", "/file/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			router.ServeHTTP(w, req)
		}

	}
	result <- re

}

func TestDownload(t *testing.T) {
	router := GetRouter()
	file, err := os.Open("./testfile/notRegister.txt")
	assert.Equal(t, nil, err)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	result := make(chan bool, 100)
	var number int
	for scanner.Scan() {
		number += 1
		usr_password := scanner.Text()
		u_p := strings.Split(usr_password, " ")
		go login(router, u_p[0], u_p[1], result)
	}
	for i := 0; i < number; i++ {
		value := <-result
		assert.Equal(t, true, value)
	}
	number = 0
	Loginmap.Range(func(key, value interface{}) bool {
		number++
		db := dao.GetDB()
		var usr model.User
		db.Where("name = ? ", value.(string)).First(&usr)
		go download(int(usr.ID), key.(string), result, router)
		return true
	})
	for i := 0; i < number; i++ {
		value := <-result
		assert.Equal(t, true, value)
	}

}

func download(userId int, token string, result chan bool, router *gin.Engine) {
	db := dao.GetDB()
	var usrfiles []model.UsrFile
	db.Where("usr_id = ?", userId).Find(&usrfiles)
	w := httptest.NewRecorder()
	index := rand.Intn(len(usrfiles))
	form := url.Values{}
	form.Add("token", token)
	form.Add("filename", usrfiles[index].FileName)
	req, _ := http.NewRequest("POST", "/file/download", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	//fmt.Printf("download content is %v", w.Body.String())
	//h := md5.New()
	downloaded, _ := ioutil.ReadAll(w.Result().Body)
	//temp := fmt.Sprintf("%x", downloaded)
	//fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("file length %v", len(downloaded)))
	downloadedMd5 := fmt.Sprintf("%x", md5.Sum(downloaded))
	var file model.File
	//fmt.Printf(temp)
	db.Where("id = ?", usrfiles[index].FileId).First(&file)
	result <- (downloadedMd5 == file.Md5sum)

}

func uploadSmall(router *gin.Engine, token string, wg *sync.WaitGroup, files []os.FileInfo) {
	defer wg.Done()
	w := httptest.NewRecorder()

	//Open file
	for i := 0; i < 20; i++ {
		file, _ := os.Open(testfiledir + files[rand.Intn(len(files))].Name())
		defer file.Close()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		//add file,token,md5sum
		part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
		io.Copy(part, file)
		writer.WriteField("token", token)
		writer.WriteField("md5sum", calculateMd5(
			filepath.Base(file.Name())))
		writer.WriteField("fragment", "no")
		//write file content
		// if err != nil {
		// 	fmt.Fprintln(gin.DefaultWriter, err.Error())
		// }
		//fmt.Fprintln(gin.DefaultWriter, writed)
		writer.Close()

		req := httptest.NewRequest("POST", "/file/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		router.ServeHTTP(w, req)
	}

}

func calculateMd5(fileName string) string {
	file, _ := os.Open(testfiledir + fileName)
	defer file.Close()
	h := md5.New()
	io.Copy(h, file)
	// if err != nil {
	// 	fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("writed %v,err:%v", writed, err.Error()))
	// }
	md5Sum := fmt.Sprintf("%x", (h.Sum(nil)[:]))
	//fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("md5Sum: %v", md5Sum))

	return md5Sum

}
