package main

import (
	"log"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"

	blogAdapter "prakhar-website-backend/adapters/blog"
	chatAdapter "prakhar-website-backend/adapters/chat"
	gemini "prakhar-website-backend/adapters/chat/gemini"
	config "prakhar-website-backend/config"
	database "prakhar-website-backend/database"
	"prakhar-website-backend/middleware"
)

func main() {
	cfg := config.LoadConfig()
	clerk.SetKey(cfg.ClerkSecretKey)
	db := database.Init(cfg.MongoDBURI)

	mux := http.NewServeMux()
	blogRepo := database.NewBlogRepo(db)
	vectorRepo := database.NewVectorRepo(db, cfg.MongoVectorDatabase, cfg.MongoVectorCollection)
	geminiClient := gemini.NewClient(cfg.GeminiAPIKey)
	rateLimiter := middleware.NewRateLimiter()

	mux.HandleFunc("GET /blogs", blogAdapter.GetBlogs(blogRepo))
	mux.Handle("POST /send-message", middleware.OptionalClerkAuth(
		rateLimiter.Middleware(
			chatAdapter.SendMessage(geminiClient, geminiClient, vectorRepo, cfg.SystemPrompt),
		),
	))

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, withMiddleware(mux)))
}

var allowedOrigins = map[string]bool{
	"https://prakhargaming.com": true,
	"http://localhost:3000":     true,
}

func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		w.Header().Set("Vary", "Origin")
		if origin := r.Header.Get("Origin"); allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
