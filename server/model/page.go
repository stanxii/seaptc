package model

//go:generate go run gogen.go -input page.go -output gen_page.go Page

type Page struct {
	Path        string `datastore:"_"`
	ContentType string `datastore:"contentType,noindex"`
	Compressed  bool   `datastore:"compressed,noindex"`
	Hash        string `datastore:"hash"`
	Data        []byte `datastore:"data,noindex"`
}
