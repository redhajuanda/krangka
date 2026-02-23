package response

type ResponseFailed struct {
	Success    bool     `json:"success" example:"false"`
	Message    string   `json:"message"`
	Data       any      `json:"data"`
	ErrorCode  string   `json:"error_code"`
	HTTPStatus int      `json:"-"`
	Metadata   Metadata `json:"metadata"`
}