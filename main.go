package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var db *sql.DB

func getTodo(c *gin.Context) {
	rows, err := db.Query("SELECT task from todo")
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("Error reading ticks: %q", err))
		return
	}
	defer rows.Close()
	for rows.Next() {
		var task string
		if err := rows.Scan(&task); err != nil {
			c.String(http.StatusInternalServerError, "Error :cant read task ::%q", err)
			return
		}

		c.String(http.StatusOK, "Task: %s \n", task)
	}
}

func main() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello World")
	})

	router.Run(":" + os.Getenv("PORT"))
}
