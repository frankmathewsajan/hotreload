package main

import (
	"fmt"
	"net/http"
)

func main() {
	// This print statement is critical. We need to see this in our terminal
	// to verify the hot-reload tool is successfully streaming the output.
	fmt.Println("[TEST SERVER] Booting up on port 8080...")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World! I am Frank!\n")
	})

	// Start the server and listen for requests
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("[TEST SERVER] Fatal error:", err)
	}
}
