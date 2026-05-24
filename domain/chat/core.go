package domain

import "context"

type EmbeddingService interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type GenerationService interface {
	Generate(ctx context.Context, prompt, systemPrompt string) (string, error)
}

type VectorRepository interface {
	Search(ctx context.Context, embedding []float64, limit int) ([]RepoDocument, error)
}
