package main

import (
	"database/sql"
	"embed"
	"flag"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed templates/**/*.tmpl
var templates embed.FS

var (
	bindAddr = flag.String("bind", ":8080", "http server bind address")
	dbPath   = flag.String("db", "todo.db", "database")
)

const title = "TODOs"

func main() {
	db, err := initDB()
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()

	router := initHTTP(db)

	router.Run(*bindAddr)
}

func initHTTP(db *sql.DB) *gin.Engine {
	repo := todoRepo{}
	router := gin.Default()
	templ := template.Must(template.New("").ParseFS(templates, "templates/**/*.tmpl"))
	router.SetHTMLTemplate(templ)
	router.GET("/", func(c *gin.Context) {
		todos, err := repo.GetTodos(db)
		if err != nil {
			log.Print(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": title,
			"tasks": todos,
		})
	})

	router.POST("/", func(c *gin.Context) {
		_, err := repo.NewTodo(db, c.PostForm("task"))
		if err != nil {
			log.Print(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		todos, err := repo.GetTodos(db)
		if err != nil {
			log.Print(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.HTML(http.StatusCreated, "components/tasks.tmpl", gin.H{
			"tasks": todos,
		})
	})
	router.DELETE("/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Params[0].Value)
		if err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		err = repo.DeleteTodo(db, id)
		if err != nil {
			log.Print(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		todos, err := repo.GetTodos(db)
		if err != nil {
			log.Print(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.HTML(http.StatusOK, "components/tasks.tmpl", gin.H{
			"tasks": todos,
		})
	})

	return router
}

func initDB() (*sql.DB, error) {
	// Create a database instance, here we'll store everything on-disk
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS todo (
		    	name TEXT NOT NULL
		    )
`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

type Todo struct {
	Id   int
	Name string
}

type todoRepo struct{}

func (tr *todoRepo) NewTodo(db *sql.DB, task string) (int, error) {
	todo := Todo{Name: task}
	_, err := db.Exec("INSERT INTO todo(name) VALUES (?)", todo.Name)
	if err != nil {
		return 0, err
	}
	return todo.Id, nil
}

func (tr *todoRepo) GetTodos(db *sql.DB) ([]Todo, error) {
	rows, err := db.Query("SELECT rowid,name FROM todo")
	if err != nil {
		return nil, err
	}
	var todos []Todo

	for rows.Next() {
		var todo Todo
		err = rows.Scan(&todo.Id, &todo.Name)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, nil
}

func (tr *todoRepo) DeleteTodo(db *sql.DB, idx int) error {
	_, err := db.Exec("DELETE FROM todo WHERE rowid = ?", idx)
	return err
}
