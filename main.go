package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/newrelic/go-agent"
	"github.com/utrack/gin-csrf"
)

var db *gorm.DB

type Todo struct {
	gorm.Model
	Title       string `gorm:"size:255"`
	Description string `gorm:"size:4095"`
	User        User
	UserId      uint
}

type User struct {
	gorm.Model
	Name     string `gorm:"size:255"`
	Email    string `gorm:"size:255"`
	Password string
}

func dbInit() {
	var todo Todo
	var user User
	//db.Model(&todo).Related(&user)
	db.DropTableIfExists(&todo)
	db.DropTableIfExists(&user)
	db.CreateTable(&todo)
	db.CreateTable(&user)

	//管理ユーザを追加
	pass, _ := toHash("password")
	adminUser := User{Name: "admin", Email: "admin@example.com", Password: pass}
	db.NewRecord(adminUser)
	db.Create(&adminUser)
	db.Save(&adminUser)
}

func newRelicMiddleware() gin.HandlerFunc {
	license := os.Getenv("NEW_RELIC_LICENSE_KEY")
	config := newrelic.NewConfig("Fierce-ocean", license)
	app, err := newrelic.NewApplication(config)
	if err != nil {
		fmt.Printf("New Relic Initialization Error")
	}
	if app == nil {
		//relicが生成できない場合(開発環境等)の場合は空の関数を返す
		return func(c *gin.Context) {
			c.Next()
		}
	}
	return func(c *gin.Context) {
		txn := app.StartTransaction(c.Request.URL.String(), c.Writer, c.Request)
		defer txn.End()
		c.Next()
	}
}

func main() {
	var err error

	db, err = gorm.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}
	defer db.Close()
	//session処理用のRedisセットアップ
	redisUrl := os.Getenv("REDIS_URL")
	var redisHost, redisPassword string
	if redisUrl != "" {
		parsedUrl, _ := url.Parse(redisUrl)
		redisPassword, _ = parsedUrl.User.Password()
		redisHost = parsedUrl.Host

	}
	store, err := sessions.NewRedisStore(10, "tcp", redisHost, redisPassword, []byte("secret"))
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

	//NewRelic設定
	router.Use(newRelicMiddleware())

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

	authorized.POST("/logout", logout)
	{
		//CSRF対策
		secure := authorized.Group("/")
		secure.Use(csrf.Middleware(csrf.Options{
			Secret:        "MyTodoSecret",
			IgnoreMethods: []string{"GET", "HEAD", "OPTIONS"},
			ErrorFunc: func(c *gin.Context) {
				c.String(400, "CSRF token mismatch")
				c.Abort()
			},
		}))
		{
			secure.GET("/todo", getTodoList)
			secure.GET("/todo/new", registerTodo)
			secure.POST("/todo", createTodo)
			secure.GET("/todo/detail/:id", getTodo)
			secure.DELETE("/todo/detail/:id", deleteTodo)
			secure.PUT("/todo/detail/:id", updateTodo)

			secure.GET("/user", getUserList)
			secure.GET("/user/new", registerUser)
			secure.POST("/user", createUser)
			secure.GET("/user/detail/:id", getUser)
			secure.DELETE("/user/detail/:id", deleteUser)
			secure.PUT("/user/detail/:id", updateUser)
		}
	}

	router.Run(":" + os.Getenv("PORT"))
}
