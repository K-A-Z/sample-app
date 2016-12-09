package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/utrack/gin-csrf"
)

func getTodoList(c *gin.Context) {
	rows, err := db.Table("todos").Select("todos.id,todos.title,users.name").Joins("join users on todos.user_id = users.id").Rows()
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("Error reading ticks: %q", err))
		return
	}
	defer rows.Close()

	var todolist []Todo
	for rows.Next() {
		var id uint
		var title, name string
		if err := rows.Scan(&id, &title, &name); err != nil {
			c.String(http.StatusInternalServerError, "Error :cant read task ::%q", err)
			return
		}
		todolist = append(todolist, Todo{Model: gorm.Model{ID: id}, Title: title, User: User{Name: name}})
	}
	fmt.Println(todolist)
	c.HTML(http.StatusOK, "list.tmpl", gin.H{
		"todo": todolist,
	})
}

func getTodo(c *gin.Context) {
	idString := c.Param("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		c.String(http.StatusInternalServerError, " Please contact the system administrator.")
	}

	var todo Todo
	db.First(&todo, id)
	fmt.Printf("Id: %d   Title:%s   Description: %s\n", todo.ID, todo.Title, todo.Description)
	c.HTML(http.StatusOK, "detail.tmpl", gin.H{
		"todo": todo,
		"csrf": csrf.GetToken(c),
	})
}

func createTodo(c *gin.Context) {
	title := c.PostForm("title")
	description := c.PostForm("description")
	session := sessions.Default(c)
	var userId uint
	fmt.Printf("userid of session: %v", session.Get("userId"))
	if u := session.Get("userId"); u != nil {
		userId = u.(uint)
	}
	fmt.Printf("addTodo: title: %s, description: %s, userId: %d", title, description, userId)
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
	paramId := c.Param("id")
	fmt.Printf("URL csrf token: %s", c.Query("_csrf"))
	if paramId == "" {
		c.String(http.StatusBadRequest, "Invalid Task id:%s", paramId)
	}
	id, err := strconv.Atoi(paramId)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid Task id:%s", paramId)
	}

	db.Delete(&Todo{}, id)
}

func updateTodo(c *gin.Context) {
	id := c.Param("id")
	title := c.Query("title")
	description := c.Query("description")
	//var currentTitle, currentDescription string
	var todo Todo
	db.First(&todo, id)
	if title != "" {
		todo.Title = title
	}
	if description != "" {
		todo.Description = description
	}
	db.Save(&todo)

}

func addTodo(title string, description string, userId uint) (id int, err error) {
	fmt.Printf("addTodo: title: %s, description: %s, userId: %d", title, description, userId)
	todo := Todo{Title: title, Description: description, UserId: userId}
	db.NewRecord(todo)
	db.Create(&todo)
	if err != nil {
		fmt.Printf("Error incrementing tick: %q", err)
		return
	}
	return
}
