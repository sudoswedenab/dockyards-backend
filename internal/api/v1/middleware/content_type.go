package middleware

import (
	"net/http"
)

type ContentType struct {
	mimeType string
}

func (t *ContentType) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", t.mimeType)

		next.ServeHTTP(w, r)
	})
}

func NewContentType(mimeType string) *ContentType {
	return &ContentType{
		mimeType: mimeType,
	}
}
