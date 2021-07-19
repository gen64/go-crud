package crud

// HTTPResponse is a base structure for all the HTTP responses from HTTP
// endpoints
type HTTPResponse struct {
	OK      int8                   `json:"ok"`
	ErrText string                 `json:"err_text"`
	Data    map[string]interface{} `json:"data"`
}

// NewHTTPResponse returns new HTTPResponse object
func NewHTTPResponse(ok int8, errText string) HTTPResponse {
	return HTTPResponse{
		OK:      ok,
		ErrText: errText,
	}
}
