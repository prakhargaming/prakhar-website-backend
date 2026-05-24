package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	domain "prakhar-website-backend/domain/blog"
	"time"
)

func GetBlogs(repo domain.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		blogs, err := repo.GetAll(ctx)
		if err != nil {
			log.Printf("GetBlogs: %v", err)
			http.Error(w, `{"error":"failed to fetch blogs"}`,
				http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(blogs)
	}
}
