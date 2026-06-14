package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	DB        string    `json:"db"`
	Version   string    `json:"version"`
}

type ItemResponse struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

func logJSON(level, msg string, fields map[string]string) {
	entry := map[string]string{
		"level":   level,
		"message": msg,
		"time":    time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range fields {
		entry[k] = v
	}
	b, _ := json.Marshal(entry)
	log.Println(string(b))
}

func connectDB() {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		os.Getenv("DB_HOST"),
		getEnv("DB_PORT", "5432"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		getEnv("DB_NAME", "appdb"),
	)

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		logJSON("error", "could not open DB", map[string]string{"error": err.Error()})
		return
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		logJSON("error", "could not ping DB", map[string]string{"error": err.Error()})
	} else {
		logJSON("info", "database connected", nil)
		initSchema()
	}
}

func initSchema() {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		logJSON("error", "schema init failed", map[string]string{"error": err.Error()})
	} else {
		logJSON("info", "schema ready", nil)
	}
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next(rw, r)
		logJSON("info", "request", map[string]string{
			"method":   r.Method,
			"path":     r.URL.Path,
			"status":   fmt.Sprintf("%d", rw.status),
			"duration": time.Since(start).String(),
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Version:   getEnv("APP_VERSION", "dev"),
		DB:        "disconnected",
	}
	if db != nil {
		if err := db.Ping(); err == nil {
			resp.DB = "connected"
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func itemsHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, `{"error":"database not available"}`, http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT id, name, created_at FROM items ORDER BY id DESC LIMIT 50")
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		items := []ItemResponse{}
		for rows.Next() {
			var item ItemResponse
			if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err == nil {
				items = append(items, item)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)

	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		var item ItemResponse
		err := db.QueryRow(
			"INSERT INTO items (name) VALUES ($1) RETURNING id, name, created_at",
			body.Name,
		).Scan(&item.ID, &item.Name, &item.CreatedAt)
		if err != nil {
			http.Error(w, `{"error":"insert failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	connectDB()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", loggingMiddleware(healthHandler))
	mux.HandleFunc("/items", loggingMiddleware(itemsHandler))
	port := getEnv("PORT", "8080")
	logJSON("info", "server starting", map[string]string{"port": port})
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		logJSON("error", "server failed", map[string]string{"error": err.Error()})
	}
}