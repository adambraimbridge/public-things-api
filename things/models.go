package things

type Things []Concept

type Concept struct {
	ID             string   `json:"id"`
	APIURL         string   `json:"apiUrl"`
	PrefLabel      string   `json:"prefLabel,omitempty"`
	Types          []string `json:"types"`
	DirectType     string   `json:"directType,omitempty"`
	Aliases        []string `json:"aliases,omitempty"`
	DescriptionXML string   `json:"descriptionXML,omitempty"`
	ImageURL       string   `json:"_imageUrl,omitempty"`
	EmailAddress   string   `json:"emailAddress,omitempty"`
	FacebookPage   string   `json:"facebookPage,omitempty"`
	TwitterHandle  string   `json:"twitterHandle,omitempty"`
	ScopeNote      string   `json:"scopeNote,omitempty"`
	ShortLabel     string   `json:"shortLabel,omitempty"`
	NarrowerThan   []Thing  `json:"narrowerThan,omitempty"`
	BroaderThan    []Thing  `json:"broaderThan,omitempty"`
	RelatedTo      []Thing  `json:"relatedTo,omitempty"`
}

type Thing struct {
	ID         string   `json:"id"`
	APIURL     string   `json:"apiUrl"`
	PrefLabel  string   `json:"prefLabel,omitempty"`
	Types      []string `json:"types"`
	DirectType string   `json:"directType,omitempty"`
}
