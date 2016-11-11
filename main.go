package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Todo struct {
	Id          int
	Title       string
	Description string
}

var db *sql.DB

func dbInit() {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS todo (id serial, title varchar(100), description varchar(1000))"); err != nil {
		fmt.Printf("Error creating database table: %q", err)
		return
	}
}

func addTodo(title string, description string) (id int, err error) {
	err = db.QueryRow("INSERT INTO todo (title, description) VALUES ($1,$2) returning id", title, description).Scan(&id)
	if err != nil {
		fmt.Printf("Error incrementing tick: %q", err)
		return
	}
	return
}

func loginForm(c *gin.Context) {
	c.HTML(http.StatusOK, "login.tmpl", nil)
}

type User struct {
	Name  string
	Email string
}

type SessionInfo struct {
	Name           interface{}
	Email          interface{}
	IsSessionAlive bool
}

func login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	if username == "" || password == "" {
		//ユーザ・パスが空ならログインに戻す
		c.HTML(http.StatusOK, "login.tmpl", gin.H{})
	}
	//ログインチェック
	isExits, User := isLoginUserExist(username, password)
	if !isExits {
		c.HTML(http.StatusOK, "login.tmpl", gin.H{})
	}
	//セッション作成
	session := sessions.Default(c)

	session.Set("name", User.Name)
	session.Set("email", User.Email)
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
	if username == "hoge" && password == "huga" {
		return true, User{Name: "testuser", Email: "hogehoge@example.com"}
	} else {
		return false, User{}
	}
}

//ログインチェック用
func isLogin(c *gin.Context) {
	var sessionInfo SessionInfo
	//セッション作成
	session := sessions.Default(c)
	sessionInfo.Name = session.Get("name")
	sessionInfo.Email = session.Get("email")
	if sessionInfo.Name == nil {
		//未ログインの場合はログイン画面に飛ばす
		loginForm(c)
	}
	c.Set("sessionInfo", sessionInfo)
}

func getTodoList(c *gin.Context) {
	isLogin(c)

	rows, err := db.Query("SELECT id, title FROM todo")
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("Error reading ticks: %q", err))
		return
	}
	defer rows.Close()

	var todolist []Todo
	for rows.Next() {
		var id int
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			c.String(http.StatusInternalServerError, "Error :cant read task ::%q", err)
			return
		}
		todolist = append(todolist, Todo{Id: id, Title: title})
	}
	fmt.Println(todolist)
	c.HTML(http.StatusOK, "list.tmpl", gin.H{
		"todo": todolist,
	})
}

func getTodo(c *gin.Context) {
	isLogin(c)
	inputId := c.Param("id")
	id, err := strconv.Atoi(inputId)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid todo number")
	}

	var title, description string
	db.QueryRow("SELECT title, description FROM todo WHERE id=$1", id).Scan(&title, &description)
	fmt.Printf("Id: %d   Title:%s   Description: %s\n", id, title, description)
	c.HTML(http.StatusOK, "detail.tmpl", gin.H{
		"todo": Todo{Id: id, Title: title, Description: description},
	})
}

func createTodo(c *gin.Context) {
	isLogin(c)
	title := c.PostForm("title")
	description := c.PostForm("description")

	id, err := addTodo(title, description)
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

	router.Run(":" + os.Getenv("PORT"))
}
