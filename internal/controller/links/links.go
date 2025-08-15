package links

import (
	"fmt"
	"net/http"
)

func GetAllLinks_Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Links endpoint reached!")
}
