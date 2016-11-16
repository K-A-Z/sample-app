package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Todo struct {
	Id          string
	Title       string
	Description string
	UserName    string
}

type User struct {
	Id    string
	Name  string
	Email string
}

var db *sql.DB

func dbInit() {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS todo (id serial, title varchar(100), description varchar(1000), userId integer)"); err != nil {
		fmt.Printf("Error creating database table: %q", err)
		return
	}
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS users (id serial, name varchar(100),email varchar(1000), password varchar(1000))"); err != nil {
		fmt.Printf("Error creating database table: %q", err)
		return
	}
	//管理ユーザを追加
	var count int
	adminEmail := "admin@example.com"
	db.QueryRow("SELECT count(*) as count FROM users WHERE email=$1", adminEmail).Scan(&count)
	if count == 0 {
		insertUser(User{Name: "admin", Email: "admin@example.com"}, "password")
	}
}

//パスワード処理
func toHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
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

func addTodo(title string, description string, userId string) (id int, err error) {
	err = db.QueryRow("INSERT INTO todo (title, description,userId) VALUES ($1,$2,$3) returning id", title, description, userId).Scan(&id)
	if err != nil {
		fmt.Printf("Error incrementing tick: %q", err)
		return
	}
	return
}

func loginForm(c *gin.Context) {
	errorMessage, _ := c.Get("loginError")
	c.HTML(http.StatusOK, "login.tmpl", gin.H{
		"ErrorMessage": errorMessage,
	})
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
	session.Set("userId", user.Id)
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
	var id, name, email, passwordHash string
	db.QueryRow("SELECT id,name,email,password FROM users WHERE email=$1", username).Scan(&id, &name, &email, &passwordHash)
	//DBのパスワードと入力されたパスワードをチェック
	if isTruePassword(password, passwordHash) {
		//認証成功
		fmt.Printf("isLoginUserExist認証成功")
		return true, User{id, name, email}
	}
	fmt.Printf("isLoginUserExist認証失敗")
	return false, User{}

}

//ログインチェック用
func isLogin(c *gin.Context) {
	//セッション作成
	session := sessions.Default(c)
	userId := session.Get("userId").(string)
	if userId == "" {
		//未ログインの場合はログイン画面に飛ばす
		loginForm(c)
	}
}

func getTodoList(c *gin.Context) {
	isLogin(c)

	rows, err := db.Query("SELECT todo.id, title ,users.name FROM todo ,users WHERE todo.userId=users.id")
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("Error reading ticks: %q", err))
		return
	}
	defer rows.Close()

	var todolist []Todo
	for rows.Next() {
		var id, title, name string
		if err := rows.Scan(&id, &title, &name); err != nil {
			c.String(http.StatusInternalServerError, "Error :cant read task ::%q", err)
			return
		}
		todolist = append(todolist, Todo{Id: id, Title: title, UserName: name})
	}
	fmt.Println(todolist)
	c.HTML(http.StatusOK, "list.tmpl", gin.H{
		"todo": todolist,
	})
}

func getTodo(c *gin.Context) {
	isLogin(c)
	id := c.Param("id")

	var title, description string
	db.QueryRow("SELECT title, description FROM todo WHERE id=$1 ", id).Scan(&title, &description)
	fmt.Printf("Id: %s   Title:%s   Description: %s\n", id, title, description)
	c.HTML(http.StatusOK, "detail.tmpl", gin.H{
		"todo": Todo{Id: id, Title: title, Description: description},
	})
}

func createTodo(c *gin.Context) {
	isLogin(c)
	title := c.PostForm("title")
	description := c.PostForm("description")
	session := sessions.Default(c)
	userId := session.Get("userId").(string)

	id, err := addTodo(title, description, userId)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: Todo is NOT created")
	}
	fmt.Printf("Insert todo:# %d ", id)

	getTodoList(c)

}
func registerTodo(c *gin.Context) {
	isLogin(c)
	c.HTML(http.StatusOK, "newtodo.tmpl", gin.H{
		"title": "TODO:New",
	})
}

func deleteTodo(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM todo WHERE id=$1", id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: Todo is NOT deleted")
	}
}

func updateTodo(c *gin.Context) {
	id := c.Param("id")
	title := c.Query("title")
	description := c.Query("description")
	var currentTitle, currentDescription string
	db.QueryRow("SELECT title description FROM todo WHERE id=$1", id).Scan(&currentTitle, &currentDescription)
	if title == "" {
		title = currentTitle
	}
	if description == "" {
		description = currentDescription
	}
	db.Exec("UPDATE todo SET title = $1, description = $2 WHERE id = $3 ", title, description, id)

}

func main() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}
	//session処理用のRedisセットアップ
	redisUrl := os.Getenv("REDIS_URL")
	var redisHost, redisPassword string
	if redisUrl != "" {
		parsedUrl, _ := url.Parse(redisUrl)
		redisPassword, _ = parsedUrl.User.Password()
		redisHost = parsedUrl.Host
	}
	store, _ := sessions.NewRedisStore(10, "tcp", redisHost, redisPassword, []byte("secret"))
	if err != nil {
		log.Fatalf("Error redis is not connected: %q", err)
	}

	//初期DBセットアップ
	dbInit()

	//ルーティング初期設定
	router := gin.Default()
	router.Use(sessions.Sessions("session", store))
	router.LoadHTMLGlob("templates/*.tmpl")
	router.Static("/assets", "./assets")

	//ルーティング
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/login")
	})
	router.GET("/login", loginForm)
	router.POST("/login", login)
	router.GET("/todo", getTodoList)
	router.GET("/todo/new", registerTodo)
	router.POST("/todo", createTodo)
	router.GET("/todo/detail/:id", getTodo)
	router.DELETE("/todo/detail/:id", deleteTodo)
	router.PUT("/todo/detail/:id", updateTodo)
	router.GET("/logout", logout)

	router.GET("/user", getUserList)
	router.GET("/user/new", registerUser)
	router.POST("/user", createUser)
	router.GET("/user/detail/:id", getUser)
	router.DELETE("/user/detail/:id", deleteUser)
	router.PUT("/user/detail/:id", updateUser)

	router.Run(":" + os.Getenv("PORT"))
}
