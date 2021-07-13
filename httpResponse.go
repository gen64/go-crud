package crud

type HTTPResponse struct {
	OK      int8                   `json:"ok"`
	ErrText string                 `json:"err_text"`
	Data    map[string]interface{} `json:"data"`
}

func NewHTTPResponse(ok int8, errText string) HTTPResponse {
	return HTTPResponse{
		OK:      ok,
		ErrText: errText,
	}
}
