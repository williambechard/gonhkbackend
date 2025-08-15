package queries

import (
	"context"
	"nhknewseasybkend/internal/graphql/resolvers"

	"github.com/graphql-go/graphql"
)

var LinksType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Link",
	Fields: graphql.Fields{
		"id":               &graphql.Field{Type: graphql.Int},
		"category_id":      &graphql.Field{Type: graphql.Int},
		"title":            &graphql.Field{Type: graphql.String},
		"description":      &graphql.Field{Type: graphql.String},
		"link":             &graphql.Field{Type: graphql.String},
		"url":              &graphql.Field{Type: graphql.String},
		"guid":             &graphql.Field{Type: graphql.String},
		"pubDate":          &graphql.Field{Type: graphql.String},
		"preview":          &graphql.Field{Type: graphql.Boolean},
		"preview_img_link": &graphql.Field{Type: graphql.String},
		"created_at":       &graphql.Field{Type: graphql.String},
		"translated":       &graphql.Field{Type: graphql.Boolean},
		"translated_date":  &graphql.Field{Type: graphql.String},
	},
})

var LinksField = &graphql.Field{
	Type: graphql.NewList(LinksType),
	Args: graphql.FieldConfigArgument{
		"limit":      &graphql.ArgumentConfig{Type: graphql.Int},
		"offset":     &graphql.ArgumentConfig{Type: graphql.Int},
		"categoryId": &graphql.ArgumentConfig{Type: graphql.Int},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		limit, _ := p.Args["limit"].(int)
		offset, _ := p.Args["offset"].(int)
		var categoryIdPtr *int
		if cat, ok := p.Args["categoryId"].(int); ok {
			categoryIdPtr = &cat
		}
		return resolvers.GetLinks(context.Background(), categoryIdPtr, limit, offset)
	},
}
