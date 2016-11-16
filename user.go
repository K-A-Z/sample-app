package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getUserList(c *gin.Context) {
	isLogin(c)

	rows, err := db.Query("SELECT id,name, email FROM users")
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("Error reading ticks: %q", err))
		return
	}
	defer rows.Close()

	var userList []User
	for rows.Next() {
		var id, name, email string
		if err := rows.Scan(&id, &name, &email); err != nil {
			c.String(http.StatusInternalServerError, "Error :cant read task ::%q", err)
			return
		}
		userList = append(userList, User{Id: id, Name: name, Email: email})
	}
	fmt.Println(userList)
	c.HTML(http.StatusOK, "userList.tmpl", gin.H{
		"userList": userList,
	})
}

func registerUser(c *gin.Context) {
	isLogin(c)
	c.HTML(http.StatusOK, "newUser.tmpl", gin.H{})
}

func createUser(c *gin.Context) {
	isLogin(c)
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
	hashPassword, err := toHash(password)
	if err != nil {
		return user, err
	}
	//データベースに登録
	var id string
	err = db.QueryRow("INSERT INTO users (name, email,password) VALUES ($1,$2,$3) returning id", user.Name, user.Email, hashPassword).Scan(&id)
	if err != nil {
		//登録に失敗したらエラーを返す
		fmt.Printf("User is not created: %q", err)
		return user, err
	}
	user.Id = id
	//UserIdを詰めて戻す
	fmt.Printf("Created user: %v", user)
	return user, nil
}

func getUser(c *gin.Context) {
	isLogin(c)
	id := c.Param("id")

	var name, email string
	db.QueryRow("SELECT name, email FROM users WHERE id=$1 ", id).Scan(&name, &email)
	fmt.Printf("Id: %s Name: %s   Email:%s   \n", id, name, email)
	c.HTML(http.StatusOK, "userDetail.tmpl", gin.H{
		"user": User{Id: id, Name: name, Email: email},
	})
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM users WHERE id=$1", id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: User is NOT deleted")
	}
}

func updateUser(c *gin.Context) {
	id := c.Param("id")
	name := c.Query("name")
	email := c.Query("email")
	var currentName, currentEmail string
	db.QueryRow("SELECT name,email FROM users WHERE id=$1", id).Scan(&currentName, &currentEmail)
	if name == "" {
		name = currentName
	}
	if email == "" {
		email = currentEmail
	}
	db.Exec("UPDATE users SET name = $1, email = $2 WHERE id = $3 ", name, email, id)
}
