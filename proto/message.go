package proto

import (
	"encoding/json"
)

type HelloMsg struct {
	Subdomain string `json:"subdomain"`
}

type HelloAckMsg struct {
	URL string `json:"url"`
}

type HelloErrMsg struct {
	Error string `json:"error"`
}

func EncodeJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func DecodeJSON[T any](data []byte) (T, error) {
	var v T
	err := json.Unmarshal(data, &v)
	return v, err
}
