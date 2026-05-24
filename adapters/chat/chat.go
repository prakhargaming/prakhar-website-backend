package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	domain "prakhar-website-backend/domain/chat"
)

type sendMessageRequest struct {
	Message string `json:"message"`
}

func SendMessage(
	embedder domain.EmbeddingService,
	generator domain.GenerationService,
	vectorRepo domain.VectorRepository,
	systemPrompt string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req sendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		embedding, err := embedder.Embed(ctx, req.Message)
		if err != nil {
			log.Printf("Embed: %v", err)
			http.Error(w, `{"error":"failed to embed query"}`, http.StatusInternalServerError)
			return
		}

		repos, err := vectorRepo.Search(ctx, embedding, 3)
		if err != nil {
			log.Printf("Search: %v", err)
			http.Error(w, `{"error":"failed to retrieve context"}`, http.StatusInternalServerError)
			return
		}

		prompt := "Context: " + strings.Join(formatRepos(repos), "\n\n") + "\n\nQuery: " + req.Message

		response, err := generator.Generate(ctx, prompt, systemPrompt)
		if err != nil {
			log.Printf("Generate: %v", err)
			http.Error(w, `{"error":"failed to generate response"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"response": response})
	}
}

func formatRepos(repos []domain.RepoDocument) []string {
	docs := make([]string, len(repos))
	for i, r := range repos {
		langs := make([]string, 0, len(r.Languages))
		for lang := range r.Languages {
			langs = append(langs, lang)
		}
		docs[i] = fmt.Sprintf(
			"# METADATA\n  Repository name: %s\n  Repository languages: %s\n  Repository topics: %s\n\n  # README:\n  %s",
			r.Name,
			strings.Join(langs, ", "),
			strings.Join(r.Topics, ", "),
			r.Readme,
		)
	}
	return docs
}
