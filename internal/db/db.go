package db

type Link struct {
	ID             int
	CategoryID     int
	Title          string
	Description    string
	Link           string
	URL            string
	GUID           string
	PubDate        string
	Preview        bool
	PreviewImgLink string
	CreatedAt      string
	Translated     bool
	TranslatedDate string
}

type Category struct {
	ID               int
	CreatedAt        string
	CategoryEnglish  string
	CategoryJapanese string
}

// Stub: GetCategoryById returns a dummy Category
func GetCategoryById(categoryID int) (*Category, error) {
	return &Category{
		ID:               categoryID,
		CreatedAt:        "2025-01-01",
		CategoryEnglish:  "News",
		CategoryJapanese: "ニュース",
	}, nil
}
