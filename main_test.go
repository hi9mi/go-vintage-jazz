package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func requestHandler(r *http.Request) *httptest.ResponseRecorder {
	db, _ := NewPostgresDB(DBConfig{Host: "localhost", Port: "5436", Username: "postgres", Password: "docker", DBName: "postgres", SSLMode: "disable"})
	repo := NewRepository(db)
	router := NewHandler(repo).initRoutes()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func TestGetRecords(t *testing.T) {
	request, _ := http.NewRequest("GET", "/albums", nil)
	w := requestHandler(request)
	if w.Code != http.StatusOK {
		t.Fatal("Expected status to be 200 but got: ", w.Code)
	}
}

func TestGetRecordByID(t *testing.T) {
	request, _ := http.NewRequest("GET", "/albums/1", nil)
	w := requestHandler(request)
	if w.Code != http.StatusOK {
		t.Fatal("Expected status to be 200 but got: ", w.Code)
	}
}

func TestDeleteRecordByID(t *testing.T) {
	request, _ := http.NewRequest("DELETE", "/albums/1", nil)
	w := requestHandler(request)
	if w.Code != http.StatusOK {
		t.Fatal("Expected status to be 200 but got: ", w.Code)
	}
}

func TestUpdateRecordByID(t *testing.T) {
	request, _ := http.NewRequest("PUT", "/albums/2", strings.NewReader(`{"title": "Updated Album"}`))
	w := requestHandler(request)
	if w.Code != http.StatusOK {
		t.Fatal("Expected status to be 200 but got: ", w.Code)
	}
}

func TestPostRecord(t *testing.T) {
	request, _ := http.NewRequest("POST", "/albums", strings.NewReader(`{"title": "New Album", "artist": "unknown", "price": 49}`))
	w := requestHandler(request)
	if w.Code != http.StatusCreated {
		t.Fatal("Expected status to be 201 but got: ", w.Code)
	}
}

func TestPostRecordBadRequest(t *testing.T) {
	request, _ := http.NewRequest("POST", "/albums", strings.NewReader(`{title: "New Album", artist: "Unknown", price: 49}`))
	w := requestHandler(request)
	if w.Code != http.StatusBadRequest {
		t.Fatal("Expected status to be 400 but got: ", w.Code)
	}
}
