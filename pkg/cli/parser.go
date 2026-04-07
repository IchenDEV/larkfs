package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func ParseJSON[T any](data []byte) (T, error) {
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("parse JSON: %w", err)
	}
	return result, nil
}

func ParseNDJSON[T any](data []byte) ([]T, error) {
	var results []T
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var item T
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, fmt.Errorf("parse NDJSON line: %w", err)
		}
		results = append(results, item)
	}
	return results, scanner.Err()
}

func StreamNDJSON[T any](r io.Reader, fn func(T) error) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var item T
		if err := json.Unmarshal(line, &item); err != nil {
			return fmt.Errorf("parse NDJSON line: %w", err)
		}
		if err := fn(item); err != nil {
			return err
		}
	}
	return scanner.Err()
}
