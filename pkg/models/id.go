package models

type NamespacedID struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`
}
