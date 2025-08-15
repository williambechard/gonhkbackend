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

// Stub: GetLinks returns dummy ArticleLinks
func GetLinks(categoryID *int, limit, offset int) ([]*Link, error) {
	links := []*Link{
		{
			ID:             1,
			CategoryID:     1,
			Title:          "Sample Article",
			Description:    "A sample article link.",
			Link:           "https://example.com/article",
			URL:            "https://example.com/article",
			GUID:           "abc123",
			PubDate:        "2025-08-13",
			Preview:        true,
			PreviewImgLink: "https://example.com/image.jpg",
			CreatedAt:      "2025-08-13",
			Translated:     false,
			TranslatedDate: "",
		},
	}
	return links, nil
}
