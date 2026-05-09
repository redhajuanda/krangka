package failure

import "github.com/redhajuanda/komon/fail"

var (
	ErrPathNotFound      = &fail.Failure{Code: "404000", Message: "There is no path that matches the given URI", HTTPStatus: 404}
	ErrNoteNotFound      = &fail.Failure{Code: "404001", Message: "Note not found", HTTPStatus: 404}
	ErrNoteAlreadyExists = &fail.Failure{Code: "409001", Message: "Note already exists", HTTPStatus: 409}
)
