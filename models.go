package main

type things []thing

type thing struct {
	ID         string   `json:"id"`
	APIURL     string   `json:"apiUrl"`
	PrefLabel  string   `json:"prefLabel,omitempty"`
	Types      []string `json:"types"`
	DirectType string   `json:"directType,omitempty"`
}
