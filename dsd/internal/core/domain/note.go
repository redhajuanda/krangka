package domain

import "time"

type Note struct {
	ID        string    `qwery:"id"`
	Title     string    `qwery:"title"`
	Content   string    `qwery:"content"`
	CreatedAt time.Time `qwery:"created_at"`
	UpdatedAt time.Time `qwery:"updated_at"`
	DeletedAt int       `qwery:"deleted_at"`
}

type NoteFilter struct {
	Search string `qwery:"search"`
}