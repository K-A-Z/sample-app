package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

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

func getTodoList(c *gin.Context) {
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
	title := c.PostForm("title")
	description := c.PostForm("description")

	id, err := addTodo(title, description)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: Todo is NOT created")
	}
	fmt.Printf("Insert todo:# %d ", id)
	c.Redirect(http.StatusMovedPermanently, "/todo")

}
func registerTodo(c *gin.Context) {
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
	dbInit()

	router := gin.Default()
	router.LoadHTMLGlob("templates/*.tmpl")
	router.Static("/assets", "./assets")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"message": "Hello World",
			"title":   "TopPage",
		})
	})
	router.GET("/todo", getTodoList)
	router.GET("/todo/new", registerTodo)
	router.POST("/todo", createTodo)
	router.GET("/todo/detail/:id", getTodo)
	router.DELETE("/todo/detail/:id", deleteTodo)
	router.PUT("/todo/detail/:id", updateTodo)

	router.Run(":" + os.Getenv("PORT"))
}
