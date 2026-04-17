package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Ankater/last-1000/internal/db"
	"github.com/Ankater/last-1000/internal/messages"
	"github.com/Ankater/last-1000/internal/tokens"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

type server struct {
	tokens   *tokens.Repository
	messages *messages.Repository
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	conn, err := db.Open()
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer conn.Close()

	s := &server{
		tokens:   &tokens.Repository{DB: conn},
		messages: &messages.Repository{DB: conn},
	}

	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	r.With(s.requireBearerAuth).Post("/messages", s.addMessage)
	r.With(s.requireBearerAuth).Get("/messages", s.getMessages)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}

func (s *server) requireBearerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(h, "Bearer ")
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		exists, err := s.tokens.TokenExists(ctx, token)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *server) addMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := s.messages.AddMessage(ctx, body.Message); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *server) getMessages(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	ctx := r.Context()
	msgs, err := s.messages.GetMessages(ctx, page)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type response struct {
		Messages []messages.Message `json:"messages"`
		Page     int                `json:"page"`
	}

	if msgs == nil {
		msgs = []messages.Message{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response{Messages: msgs, Page: page})
}
