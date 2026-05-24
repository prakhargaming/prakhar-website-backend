package database

import (
	"context"
	domain "prakhar-website-backend/domain/blog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type BlogRepo struct {
	coll *mongo.Collection
}

func NewBlogRepo(client *mongo.Client) *BlogRepo {
	return &BlogRepo{coll: client.Database("Prakharbase").Collection("blogs")}
}

func (r *BlogRepo) GetAll(ctx context.Context) ([]domain.Blog, error) {
	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})
	cursor, err := r.coll.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	type rawDoc struct {
		domain.Blog `bson:",inline"`
		RawID       bson.ObjectID `bson:"_id"`
	}

	var docs []rawDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	blogs := make([]domain.Blog, len(docs))
	for i, d := range docs {
		blogs[i] = d.Blog
		blogs[i].ID = d.RawID.Hex()
	}
	return blogs, nil
}
