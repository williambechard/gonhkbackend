package category

type Category struct {
	ID               int
	CreatedAt        string
	CategoryEnglish  string
	CategoryJapanese string
}

// GetCategoryById returns a dummy Category (stub)
func GetCategoryById(categoryID int) (*Category, error) {
	return &Category{
		ID:               categoryID,
		CreatedAt:        "2025-01-01",
		CategoryEnglish:  "News",
		CategoryJapanese: "ニュース",
	}, nil
}
