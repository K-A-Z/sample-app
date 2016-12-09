package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/utrack/gin-csrf"
)

func getUserList(c *gin.Context) {

	//rows, err := db.Query("SELECT id,name, email FROM users")
	var users []User
	db.Find(&users)
	fmt.Println(users)
	c.HTML(http.StatusOK, "userList.tmpl", gin.H{
		"userList": users,
	})
}

func registerUser(c *gin.Context) {
	c.HTML(http.StatusOK, "newUser.tmpl", gin.H{
		"csrf": csrf.GetToken(c),
	})

}

func createUser(c *gin.Context) {
	name := c.PostForm("name")
	email := c.PostForm("email")
	password := c.PostForm("password")

	user := User{Name: name, Email: email}
	createdUser, err := insertUser(user, password)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: User is NOT created")
	}
	fmt.Printf("Insert User:# %v ", createdUser)

	getUserList(c)

}

func insertUser(user User, password string) (User, error) {
	//バリデーション
	//パスワードのハッシュ化
	user.Password, _ = toHash(password)

	//データベースに登録
	db.NewRecord(user)
	errs := db.Create(&user).GetErrors()
	for _, err := range errs {
		if err != nil {
			//登録に失敗したらエラーを返す
			fmt.Printf("User is not created: %q", err)
			return user, err
		}
	}
	fmt.Printf("Created user: %v", user)
	return user, nil
}

func getUser(c *gin.Context) {
	idString := c.Param("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		c.String(http.StatusInternalServerError, " Please contact the system administrator.")
	}

	var user User
	db.First(&user, id)
	fmt.Printf("Id: %d Name: %s   Email:%s   \n", user.ID, user.Name, user.Email)
	c.HTML(http.StatusOK, "userDetail.tmpl", gin.H{
		"csrf": csrf.GetToken(c),
		"user": user,
	})
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")
	errs := db.Delete(&User{}, id).GetErrors()
	for _, err := range errs {
		if err != nil {
			c.String(http.StatusInternalServerError, "Error: User is NOT deleted")
		}
	}
}

func updateUser(c *gin.Context) {
	id := c.Param("id")
	name := c.Query("name")
	email := c.Query("email")
	var user User
	db.First(&user, id)
	if name != "" {
		user.Name = name
	}
	if email != "" {
		user.Email = email
	}
	db.Update(user)
}
