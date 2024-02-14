package formatter

type ResponseFormat struct {
	// The format of the response.
	// If you want your value to be returned as a json object, specify the type as "json_object".
	Type string `json:"type"`
}
