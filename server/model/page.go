package model

import "cloud.google.com/go/datastore"

//go:generate go run gogen.go -input page.go -output gen_page.go Page

type Page struct {
	Path        string `datastore:"_"`
	ContentType string `datastore:"contentType,noindex"`
	Compressed  bool   `datastore:"compressed,noindex"`
	Hash        string `datastore:"hash"`
	Data        []byte `datastore:"data,noindex"`
}

func (page *Page) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(page, ps)
}

func (page *Page) Save() ([]datastore.Property, error) {
	ps, err := datastore.SaveStruct(page)
	return ps, err
}

func (page *Page) LoadKey(k *datastore.Key) error {
	page.Path = k.Name
	return nil
}
