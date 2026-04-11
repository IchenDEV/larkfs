package cli

import "encoding/json"

type WrappedData[T any] struct {
	Data T `json:"data"`
}

func ParseWrappedData[T any](data []byte) (T, error) {
	var wrapped WrappedData[T]
	if err := json.Unmarshal(data, &wrapped); err != nil {
		var zero T
		return zero, err
	}
	return wrapped.Data, nil
}
