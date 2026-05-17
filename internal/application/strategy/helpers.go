package strategy

import (
	"encoding/json"

	"github.com/3122380051/golang-microservices/internal/domain"
)

func mustJSON(v any) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}

func mustJSONSignal(v domain.Signal) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}
