package main

import (
	"errors"
	"fmt"
	"net/http"

	"database/sql"
	"github.com/gin-gonic/gin"

	_ "github.com/lib/pq"
)

type Records interface {
	Create(PostRecordInput) (Record, error)
	Read() ([]Record, error)
	ReadOne(string) (Record, error)
	Update(string, UpdateRecordInput) (Record, error)
	Delete(string) (string, error)
}

type Handlers interface {
	getRecords(c *gin.Context)
	postRecord(c *gin.Context)
	getRecordByID(c *gin.Context)
	deleteRecordByID(c *gin.Context)
	updateRecordByID(c *gin.Context)
}

type Record struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Price  int    `json:"price"`
}

type PostRecordInput struct {
	Title  string `json:"title" binding:"required"`
	Artist string `json:"artist" binding:"required"`
	Price  int    `json:"price" binding:"required"`
}

type UpdateRecordInput struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Price  int    `json:"price"`
}

type Handler struct {
	repo *Repository
}

type Repository struct {
	db *sql.DB
}

type DBConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	DBName   string
	SSLMode  string
}

func main() {
	db, err := NewPostgresDB(DBConfig{Host: "localhost", Port: "5436", Username: "postgres", Password: "docker", DBName: "postgres", SSLMode: "disable"})
	if err != nil {
		fmt.Printf("Failed to initialize db: %s", err.Error())
	} else {
		repo := NewRepository(db)
		handler := NewHandler(repo)
		defer db.Close()
		fmt.Println("Successfully connected to db!!!")
		r := handler.initRoutes()
		r.Run("localhost:8080")
	}
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo}
}

func (h *Handler) initRoutes() *gin.Engine {
	r := gin.Default()
	r.GET("/albums", h.getRecords)
	r.POST("/albums", h.postRecord)
	r.GET("/albums/:id", h.getRecordByID)
	r.DELETE("/albums/:id", h.deleteRecordByID)
	r.PUT("/albums/:id", h.updateRecordByID)
	return r
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db}
}

func NewPostgresDB(cfg DBConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.DBName, cfg.Password, cfg.SSLMode))
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (r *Repository) Create(a PostRecordInput) (Record, error) {
	qry := "INSERT INTO records (title, artist, price) VALUES ($1, $2, $3) RETURNING *"
	row, err := r.db.Query(qry, a.Title, a.Artist, a.Price)
	if err != nil {
		return Record{}, err
	}
	defer row.Close()
	var id, title, artist string
	var price int
	for row.Next() {
		if err := row.Scan(&id, &title, &artist, &price); err != nil {
			return Record{}, err
		}
	}
	return Record{ID: id, Title: title, Artist: artist, Price: price}, nil
}

func (r *Repository) Read() ([]Record, error) {
	fmt.Printf("Db value: %v", r.db)
	qry := "SELECT * FROM records"
	rows, err := r.db.Query(qry)
	var records []Record
	if err != nil {
		return []Record{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, title, artist string
		var price int
		if err := rows.Scan(&id, &title, &artist, &price); err != nil {
			return []Record{}, err
		}
		record := Record{ID: id, Title: title, Artist: artist, Price: price}
		records = append(records, record)
	}
	rows.Columns()
	return records, nil
}

func (r *Repository) ReadOne(id string) (Record, error) {
	qry := "SELECT * FROM records WHERE id = $1"
	row, err := r.db.Query(qry, id)
	if err != nil {
		return Record{}, errors.New("record not found")
	}
	defer row.Close()
	var title, artist string
	var price int
	for row.Next() {
		if err := row.Scan(&id, &title, &artist, &price); err != nil {
			return Record{}, err
		}
	}
	return Record{ID: id, Title: title, Artist: artist, Price: price}, nil
}

func (r *Repository) Update(id string, newRecord UpdateRecordInput) (Record, error) {
	qry := "UPDATE records SET title = $1, artist = $2, price = $3 WHERE id = $4 RETURNING *"
	row, err := r.db.Query(qry, newRecord.Title, newRecord.Artist, newRecord.Price, id)
	if err != nil {
		return Record{}, err
	}
	defer row.Close()
	var title, artist string
	var price int
	for row.Next() {
		if err := row.Scan(&id, &title, &artist, &price); err != nil {
			return Record{}, err
		}
	}
	return Record{ID: id, Title: title, Artist: artist, Price: price}, nil
}

func (r *Repository) Delete(id string) (string, error) {
	qry := "DELETE FROM records WHERE id = $1"
	_, err := r.db.Query(qry, id)
	if err != nil {
		return "", err
	}
	return "record successfully deleted", nil
}

func (h *Handler) getRecords(c *gin.Context) {
	r, err := h.repo.Read()
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, r)
}

func (h *Handler) postRecord(c *gin.Context) {
	var inputRecord PostRecordInput
	if err := c.BindJSON(&inputRecord); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	r, err := h.repo.Create(inputRecord)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusCreated, r)
}

func (h *Handler) getRecordByID(c *gin.Context) {
	id := c.Param("id")
	r, err := h.repo.ReadOne(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, r)
}

func (h *Handler) deleteRecordByID(c *gin.Context) {
	id := c.Param("id")
	r, err := h.repo.Delete(id)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"message": r})
}

func (h *Handler) updateRecordByID(c *gin.Context) {
	id := c.Param("id")
	var inputRecord UpdateRecordInput
	if err := c.BindJSON(&inputRecord); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	r, err := h.repo.Update(id, inputRecord)
	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": err.Error()})
	}
	c.IndentedJSON(http.StatusOK, r)
}
