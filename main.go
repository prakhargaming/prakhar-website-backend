package main

import (
	"log"
	"net/http"

	blogAdapter "prakhar-website-backend/adapters/blog"
	chatAdapter "prakhar-website-backend/adapters/chat"
	gemini "prakhar-website-backend/adapters/chat/gemini"
	config "prakhar-website-backend/config"
	database "prakhar-website-backend/database"
)

func main() {
	cfg := config.LoadConfig()
	db := database.Init(cfg.MongoDBURI)

	mux := http.NewServeMux()
	blogRepo := database.NewBlogRepo(db)
	vectorRepo := database.NewVectorRepo(db, cfg.MongoVectorDatabase, cfg.MongoVectorCollection)
	geminiClient := gemini.NewClient(cfg.GeminiAPIKey)

	mux.HandleFunc("GET /blogs", blogAdapter.GetBlogs(blogRepo))
	mux.HandleFunc("POST /send-message", chatAdapter.SendMessage(geminiClient, geminiClient, vectorRepo, cfg.SystemPrompt))

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, withMiddleware(mux)))
}

func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
