package main

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

//パスワード処理
func toHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

func loginForm(c *gin.Context) {
	errorMessage, _ := c.Get("loginError")
	c.HTML(http.StatusOK, "login.tmpl", gin.H{
		"ErrorMessage": errorMessage,
	})
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		//セッション情報がある場合にはログインをしているとは判定
		session := sessions.Default(c)
		userId := session.Get("userId")
		if userId == nil {
			//未ログインの場合はログイン画面に飛ばす
			loginForm(c)
		}
		fmt.Printf("Authorized User Session:: userid:%d username: %s ", session.Get("userId"), session.Get("name"))
		c.Next()
	}
}

func login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	if username == "" || password == "" {
		//ユーザ・パスが空ならログインに戻す
		c.Set("loginError", "ユーザ名またはパスワードが間違っています。")
		loginForm(c)
	}
	//ログインチェック
	isExits, user := isLoginUserExist(username, password)
	fmt.Printf("userexists%v username:%s", isExits, user.Name)
	if !isExits {
		c.Set("loginError", "ユーザ名またはパスワードが間違っています。")
		loginForm(c)
	}
	//セッション作成
	session := sessions.Default(c)

	session.Set("name", user.Name)
	session.Set("email", user.Email)
	session.Set("userId", user.ID)
	session.Save()

	getTodoList(c)

}

func logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	loginForm(c)
}

func isLoginUserExist(username, password string) (bool, User) {
	var user User
	db.Where("email = ?", username).First(&user)
	//DBのパスワードと入力されたパスワードをチェック
	if isTruePassword(password, user.Password) {
		//認証成功
		fmt.Printf("isLoginUserExist認証成功")

		return true, user
	}
	fmt.Printf("isLoginUserExist認証失敗")
	return false, User{}

}

func isTruePassword(password, passwordHash string) bool {
	decodedPasswordHash, _ := hex.DecodeString(passwordHash)
	err := bcrypt.CompareHashAndPassword(decodedPasswordHash, []byte(password))
	if err == nil {
		//認証に成功した場合にはerrが帰らないので認証成功としてtrueを返す
		return true
	}
	return false
}
