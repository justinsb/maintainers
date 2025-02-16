package model

// KEP represents a Kubernetes Enhancement Proposal
type KEP struct {
	// Path is the path to the KEP file, relative to the repo base
	Path string `json:"path"`

	// Title is the title of the KEP
	Title string `json:"title"`

	// Number is the number of the KEP
	Number string `json:"number"`

	// Authors are the authors of the KEP
	Authors []string `json:"authors"`

	// Status is the status of the KEP
	Status string `json:"status"`

	// TextURL is the URL to the KEP README.md file
	TextURL string `json:"textURL"`

	// TextContents is the contents of the KEP README.md file
	TextContents string `json:"-"`
}
