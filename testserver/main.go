package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("[TEST SERVER] Booting up on port 8080")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Hot Reload</title>
		</head>
		<body>
			<h1>Go Live Server!!!!!</h1>
			<p>Modify this text in testserver/main.go and hit save</p>
			
			<script>
				// The browser pings the server every second.
				// If the ping fails (CLI tool killed the server to rebuild it),
				// the browser rapidly checks until the server is back up, then refreshes
				setInterval(() => {
					fetch('/').catch(() => {
						let check = setInterval(() => {
							fetch('/').then(() => {
								clearInterval(check);
								window.location.reload();
							}).catch(() => {});
						}, 200);
					});
				}, 1000);
			</script>
		</body>
		</html>
		`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("[TEST SERVER] Fatal error:", err)
	}
}
