package main

import (
	"io"
	"net/http"
	"os"
)

func readAll(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, io.EOF
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	_ = r.Body.Close()
	return data, nil
}

func openPreviewFile(full string) (*os.File, error) {
	return os.Open(full)
}
