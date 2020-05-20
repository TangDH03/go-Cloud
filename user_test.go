package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var Loginmap sync.Map

func TestInitOK(t *testing.T) {
	router := GetRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `{"message":"OK"}`, w.Body.String())

}

func TestReigister(t *testing.T) {
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
		go register(router, u_p[0], u_p[1], result)
	}
	for i := 0; i < number; i++ {
		value := <-result
		assert.Equal(t, true, value)
	}

	refile, err := os.Open("./testfile/registered.txt")
	assert.Equal(t, nil, err)
	defer refile.Close()
	scanner = bufio.NewScanner(refile)

	number = 0
	for scanner.Scan() {
		number += 1
		usr_password := scanner.Text()
		u_p := strings.Split(usr_password, " ")
		go userExistRegister(router, u_p[0], u_p[1], result)

	}

	for i := 0; i < number; i++ {
		value := <-result
		assert.Equal(t, true, value)
	}

}

func register(router *gin.Engine, usr string, passwd string, result chan bool) {
	w := httptest.NewRecorder()
	form := url.Values{}
	form.Add("name", usr)
	form.Add("password", passwd)

	req, _ := http.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	const success = `\{"message":"success","token":.*\}`
	validCode := regexp.MustCompile(success)
	matched := validCode.MatchString(w.Body.String())

	if w.Code == 200 && matched {
		result <- true
		return
	}
	result <- false
}
func userExistRegister(router *gin.Engine, usr string, passwd string, result chan bool) {
	w := httptest.NewRecorder()
	form := url.Values{}
	form.Add("name", usr)
	form.Add("password", passwd)

	req, _ := http.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	const success = `\{"message":"user exist","token".*\}`
	validCode := regexp.MustCompile(success)
	matched := validCode.MatchString(w.Body.String())
	if w.Code == 200 && matched {
		result <- true
		return
	}
	result <- false

}

func TestLogin(t *testing.T) {
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
	refile, err := os.Open("./testfile/registered.txt")
	assert.Equal(t, nil, err)
	defer refile.Close()
	scanner = bufio.NewScanner(refile)

	number = 0
	for scanner.Scan() {
		number += 1
		usr_password := scanner.Text()
		u_p := strings.Split(usr_password, " ")
		go loginFail(router, u_p[0], u_p[1]+"1", result)

	}

	for i := 0; i < number; i++ {
		value := <-result
		assert.Equal(t, true, value)
	}
	hastoken := make(chan bool, 10)
	notoken := make(chan bool, 10)
	hasexpiretoken := make(chan bool, 10)
	Loginmap.Range(func(key, value interface{}) bool {
		go loginWithToken(router, key.(string), hastoken)
		go loginWithNoToken(router, "", notoken)
		go loginWithExpireToken(router, key.(string), hasexpiretoken)
		return true
	})
	for i := 0; i < number*3; i++ {
		select {
		case value := <-hastoken:
			assert.Equal(t, true, value)
		case value := <-notoken:
			assert.Equal(t, true, value)
		case value := <-hasexpiretoken:
			assert.Equal(t, true, value)
		}
	}
}
func login(router *gin.Engine, name string, passwd string, result chan bool) {
	w := httptest.NewRecorder()
	form := url.Values{}
	form.Add("name", name)
	form.Add("password", passwd)

	req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	const success = `\{"message":"success","token":.*\}`
	validCode := regexp.MustCompile(success)
	matched := validCode.MatchString(w.Body.String())
	const tokenP = `\{"message":"success","token":"(.*)"\}`
	contentPattern := regexp.MustCompile(tokenP)
	//	fmt.Fprintln(gin.DefaultWriter, w.Body.String())

	//	fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("matched %v", matched))
	tokenSlice := contentPattern.FindStringSubmatch(w.Body.String())
	Loginmap.Store(tokenSlice[1], name)
	if w.Code == 200 && matched {
		result <- true
		return
	}
	result <- false
}
func loginFail(router *gin.Engine, name string, passwd string, result chan bool) {
	w := httptest.NewRecorder()
	form := url.Values{}
	form.Add("name", name)
	form.Add("password", passwd)

	req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	const success = `\{"message":"fail","token":.*\}`
	validCode := regexp.MustCompile(success)
	matched := validCode.MatchString(w.Body.String())

	if w.Code == 200 && matched {
		result <- true
		return
	}
	result <- false

}

func loginWithToken(router *gin.Engine, token string, result chan bool) {
	w := httptest.NewRecorder()
	form := url.Values{}
	form.Add("token", token)
	validName, _ := Loginmap.Load(token)
	req, _ := http.NewRequest("POST", "/file/test", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	const success = `\{"message":"success","name":.*\}`
	fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("token : %v", token))
	validCode := regexp.MustCompile(success)
	matched := validCode.MatchString(w.Body.String())
	fmt.Fprintln(gin.DefaultWriter, w.Body.String())
	const nPattern = `\{"message":"success","name":"(.*)"\}`
	namePattern := regexp.MustCompile(nPattern)
	nameSlice := namePattern.FindStringSubmatch(w.Body.String())
	matched = matched && (validName.(string) == nameSlice[1])
	//	fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("matched %v", matched))
	if w.Code == 200 && matched {
		result <- true
		return
	}
	result <- false

}
func loginWithNoToken(router *gin.Engine, token string, result chan bool) {
	w := httptest.NewRecorder()
	form := url.Values{}
	form.Add("token", "")

	req, _ := http.NewRequest("POST", "/file/test", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	const success = `\{"message":"UnLogin"\}`
	validCode := regexp.MustCompile(success)
	matched := validCode.MatchString(w.Body.String())
	fmt.Fprintln(gin.DefaultWriter, w.Body.String())

	//	fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("matched %v", matched))
	if w.Code == 200 && matched {
		result <- true
		return
	}
	result <- false

}
func loginWithExpireToken(router *gin.Engine, token string, result chan bool) {
	w := httptest.NewRecorder()
	form := url.Values{}
	form.Add("token", token+"21")

	req, _ := http.NewRequest("POST", "/file/test", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	const success = `\{"message":"token expire"\}`
	validCode := regexp.MustCompile(success)
	matched := validCode.MatchString(w.Body.String())
	//	fmt.Fprintln(gin.DefaultWriter, w.Body.String())

	//	fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("matched %v", matched))
	if w.Code == 200 && matched {
		result <- true
		return
	}
	result <- false

}
