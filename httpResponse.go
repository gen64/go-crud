package crud

type HTTPResponse struct {
	OK      bool                   `json:"ok"`
	ErrText string                 `json:"err_text"`
	Data    map[string]interface{} `json:"data"`
}

func NewHTTPResponse(ok bool, errText string) HTTPResponse {
	return HTTPResponse{
		OK:      ok,
		ErrText: errText,
	}
}
