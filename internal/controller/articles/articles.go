package articles

import (
	"fmt"
	"net/http"
)

func GetAllArticles_Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Articles endpoint reached!")
}
