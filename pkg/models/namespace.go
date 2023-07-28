package models

import "net/http"

type Namespace struct {
	Namespace string `json:"namespace" yaml:"namespace"`
}

func (n *Namespace) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
