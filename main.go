package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	_ "github.com/lib/pq"
)

type Records interface {
	Create(PostRecordInput) (string, error)
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
	if err := InitConfig(); err != nil {
		log.Fatalf("Failed to read config: %s", err.Error())
	}
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load .env file: %s", err.Error())
	}
	db, err := NewPostgresDB(DBConfig{
		Host:     viper.GetString("db.host"),
		Port:     viper.GetString("db.port"),
		Username: viper.GetString("db.username"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   viper.GetString("db.dbname"),
		SSLMode:  viper.GetString("db.sslmode")})
	if err != nil {
		log.Fatalf("Failed to initialize db: %s", err.Error())
	} else {
		defer db.Close()
		repo := NewRepository(db)
		handler := NewHandler(repo)
		fmt.Println("Successfully connected to db!!!")
		r := handler.initRoutes()
		r.Run(fmt.Sprintf("localhost:%s", viper.GetString("port")))
	}
}

func InitConfig() error {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	return viper.ReadInConfig()
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

func (r *Repository) Create(rcrd PostRecordInput) (string, error) {
	var id string
	qry := "INSERT INTO records (title, artist, price) VALUES ($1, $2, $3) RETURNING id"
	row := r.db.QueryRow(qry, rcrd.Title, rcrd.Artist, rcrd.Price)
	err := row.Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to get last insert id: %s", err.Error())
	}
	return id, nil
}

func (r *Repository) Read() ([]Record, error) {
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
	return records, nil
}

func (r *Repository) ReadOne(id string) (Record, error) {
	var rcrd Record
	qry := "SELECT * FROM records WHERE id = $1"
	row := r.db.QueryRow(qry, id)
	if err := row.Scan(&rcrd.ID, &rcrd.Title, &rcrd.Artist, &rcrd.Price); err != nil {
		if err == sql.ErrNoRows {
			return rcrd, fmt.Errorf("record with id %s not found", id)
		}
		return rcrd, fmt.Errorf("failed to get record with id %s: %s", id, err.Error())
	}
	return rcrd, nil
}

func (r *Repository) Update(id string, newRecord UpdateRecordInput) (Record, error) {
	var rcrd Record
	setValues := make([]string, 0)
	if newRecord.Title != "" {
		setValues = append(setValues, fmt.Sprintf("title = '%s'", newRecord.Title))
	}
	if newRecord.Artist != "" {
		setValues = append(setValues, fmt.Sprintf("artist = '%s'", newRecord.Artist))
	}
	if newRecord.Price != 0 {
		setValues = append(setValues, fmt.Sprintf("price = %d", newRecord.Price))
	}
	setQuery := strings.Join(setValues, ", ")
	qry := fmt.Sprintf("UPDATE records SET %s WHERE id = %s RETURNING *", setQuery, id)
	row := r.db.QueryRow(qry)
	if err := row.Scan(&rcrd.ID, &rcrd.Title, &rcrd.Artist, &rcrd.Price); err != nil {
		if err == sql.ErrNoRows {
			return rcrd, fmt.Errorf("record with id %s not found", id)
		}
		return rcrd, fmt.Errorf("failed to get record with id %s: %s", id, err.Error())
	}
	return rcrd, nil
}

func (r *Repository) Delete(id string) (string, error) {
	qry := "DELETE FROM records WHERE id = $1 RETURNING id"
	row := r.db.QueryRow(qry, id)
	if err := row.Scan(&id); err == sql.ErrNoRows {
		return "", fmt.Errorf("record with id %s not found", id)
	}
	return id, nil
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
	c.IndentedJSON(http.StatusCreated, gin.H{"id": r})
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
	c.IndentedJSON(http.StatusOK, gin.H{"id": r})
}

func (h *Handler) updateRecordByID(c *gin.Context) {
	var inputRecord UpdateRecordInput
	id := c.Param("id")
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
