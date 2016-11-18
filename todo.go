package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/utrack/gin-csrf"
)

func getTodoList(c *gin.Context) {
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
	id := c.Param("id")

	var title, description string
	db.QueryRow("SELECT title, description FROM todo WHERE id=$1 ", id).Scan(&title, &description)
	fmt.Printf("Id: %s   Title:%s   Description: %s\n", id, title, description)
	c.HTML(http.StatusOK, "detail.tmpl", gin.H{
		"todo": Todo{Id: id, Title: title, Description: description},
	})
}

func createTodo(c *gin.Context) {
	title := c.PostForm("title")
	description := c.PostForm("description")
	session := sessions.Default(c)
	var userId string
	if u := session.Get("userId"); u != nil {
		userId = u.(string)
	}

	id, err := addTodo(title, description, userId)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: Todo is NOT created")
	}
	fmt.Printf("Insert todo:# %d ", id)

	getTodoList(c)

}
func registerTodo(c *gin.Context) {
	c.HTML(http.StatusOK, "newtodo.tmpl", gin.H{
		"csrf":  csrf.GetToken(c),
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

func addTodo(title string, description string, userId string) (id int, err error) {
	err = db.QueryRow("INSERT INTO todo (title, description,userId) VALUES ($1,$2,$3) returning id", title, description, userId).Scan(&id)
	if err != nil {
		fmt.Printf("Error incrementing tick: %q", err)
		return
	}
	return
}
