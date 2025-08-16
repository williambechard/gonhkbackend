package resolvers

import (
	"context"
	"net/http"
	linksdb "nhknewseasybkend/internal/db/links"
	"nhknewseasybkend/internal/util"
)

// LinkResolver provides resolver methods for Link type
type LinkResolver struct{}

// GetLinks returns a list of Link objects
func GetLinks(ctx context.Context, categoryID *int, limit, offset int) ([]*linksdb.Link, error) {
	util.Log("Fetching links with categoryID filter", util.LogTypeLog)
	realClient := &http.Client{}
	links, err := linksdb.GetLinks(realClient, limit, offset, categoryID)
	return links, err
}
