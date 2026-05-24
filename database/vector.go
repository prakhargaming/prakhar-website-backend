package database

import (
	"context"
	domain "prakhar-website-backend/domain/chat"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type VectorRepo struct {
	coll *mongo.Collection
}

func NewVectorRepo(client *mongo.Client, database, collection string) *VectorRepo {
	return &VectorRepo{coll: client.Database(database).Collection(collection)}
}

func (r *VectorRepo) Search(ctx context.Context, embedding []float64, limit int) ([]domain.RepoDocument, error) {
	pipeline := bson.A{
		bson.D{{Key: "$vectorSearch", Value: bson.D{
			{Key: "index", Value: "vector_index"},
			{Key: "queryVector", Value: embedding},
			{Key: "path", Value: "embedding"},
			{Key: "exact", Value: true},
			{Key: "limit", Value: limit},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "name", Value: 1},
			{Key: "readme", Value: 1},
			{Key: "topics", Value: 1},
			{Key: "languages", Value: 1},
		}}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	type repoDoc struct {
		Name      string           `bson:"name"`
		Readme    string           `bson:"readme"`
		Topics    []string         `bson:"topics"`
		Languages map[string]int64 `bson:"languages"`
	}

	var docs []repoDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	results := make([]domain.RepoDocument, len(docs))
	for i, d := range docs {
		results[i] = domain.RepoDocument(d)
	}
	return results, nil
}
