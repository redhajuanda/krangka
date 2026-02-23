package domain

import "time"

type Todo struct {
	ID          string    `qwery:"id"`
	Title       string    `qwery:"title"`
	Description string    `qwery:"description"`
	Done        bool      `qwery:"done"`
	CreatedAt   time.Time `qwery:"created_at"`
	UpdatedAt   time.Time `qwery:"updated_at"`
	DeletedAt   int       `qwery:"deleted_at"`
}

type TodoFilter struct {
	Search string `qwery:"search"`
	IsDone *bool  `qwery:"is_done"`
}
