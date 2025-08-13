package service

import (
	"fmt"
	"net/http"
)

func LinksHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Links endpoint reached!")
}
