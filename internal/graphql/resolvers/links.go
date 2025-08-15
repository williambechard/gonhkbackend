package resolvers

import (
	"context"
	categorydb "nhknewseasybkend/internal/db/category"
	linksdb "nhknewseasybkend/internal/db/links"
	"nhknewseasybkend/internal/util"
)

// LinkResolver provides resolver methods for Link type
type LinkResolver struct{}

// Category resolves the category field for Link
func (r *LinkResolver) Category(ctx context.Context, parent *linksdb.Link) (*categorydb.Category, error) {
	util.Log("Resolving category for link ID: "+string(rune(parent.ID)), util.LogTypeLog)
	return categorydb.GetCategoryById(parent.CategoryID)
}

// GetLinks returns a list of Link objects
func GetLinks(ctx context.Context, categoryID *int, limit, offset int) ([]*linksdb.Link, error) {
	util.Log("Fetching links with categoryID filter", util.LogTypeLog)
	links, err := linksdb.GetLinks(limit, offset, categoryID)
	return links, err
}
