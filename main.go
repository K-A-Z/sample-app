package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
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

	//セッション設定
	router.Use(sessions.Sessions("session", store))

	//セキュリティ設定
	router.Use(secure.Secure(secure.Options{
		//AllowedHosts:          []string{"example.com", "ssl.example.com"},
		//SSLRedirect: true,
		//SSLHost:               "ssl.example.com",
		//SSLProxyHeaders:      map[string]string{"X-Forwarded-Proto": "https"},
		STSSeconds:           315360000,
		STSIncludeSubdomains: true,
		FrameDeny:            true,
		ContentTypeNosniff:   true,
		BrowserXssFilter:     true,
		//ContentSecurityPolicy: "default-src 'self'",
	}))

	router.LoadHTMLGlob("templates/*.tmpl")
	router.Static("/assets", "./assets")

	//ルーティング
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/login")
	})
	router.GET("/login", loginForm)
	router.POST("/login", login)

	authorized := router.Group("/")
	authorized.Use(AuthRequired())
	{
		authorized.GET("/todo", getTodoList)
		authorized.GET("/todo/new", registerTodo)
		authorized.POST("/todo", createTodo)
		authorized.GET("/todo/detail/:id", getTodo)
		authorized.DELETE("/todo/detail/:id", deleteTodo)
		authorized.PUT("/todo/detail/:id", updateTodo)
		authorized.POST("/logout", logout)

		authorized.GET("/user", getUserList)
		authorized.GET("/user/new", registerUser)
		authorized.POST("/user", createUser)
		authorized.GET("/user/detail/:id", getUser)
		authorized.DELETE("/user/detail/:id", deleteUser)
		authorized.PUT("/user/detail/:id", updateUser)
	}

	router.Run(":" + os.Getenv("PORT"))
}
