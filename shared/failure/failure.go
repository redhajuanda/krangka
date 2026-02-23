package failure

import "github.com/redhajuanda/komon/fail"

var (
	ErrTodoNotFound      = &fail.Failure{Code: "404001", Message: "Todo not found", HTTPStatus: 404}
	ErrTodoAlreadyExists = &fail.Failure{Code: "409001", Message: "Todo already exists", HTTPStatus: 409}

	ErrNoteNotFound      = &fail.Failure{Code: "404002", Message: "Note not found", HTTPStatus: 404}
	ErrNoteAlreadyExists = &fail.Failure{Code: "409002", Message: "Note already exists", HTTPStatus: 409}
)
