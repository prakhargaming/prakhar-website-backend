package domain

import "time"

type Blog struct {
	ID      string    `bson:"-" json:"_id"`
	Title   string    `bson:"title"         json:"title"`
	Content string    `bson:"content"       json:"content"`
	Author  string    `bson:"author"        json:"author"`
	Date    time.Time `bson:"date"          json:"date"`
	Tags    []string  `bson:"tags"          json:"tags"`
	Locked  bool      `bson:"locked"        json:"locked"`
	Image   string    `bson:"image"         json:"image,omitempty"`
}
