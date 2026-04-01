package main

import (
	"flag"
	"log"
	"net/http"

	dashboard "github.com/ialexeze/orkestra-ui/pkg"
)

func main() {
    orkestraURL := flag.String("u", "http://localhost:8080", "URL of the Orkestra runtime")
    port := flag.String("p", "8081", "Port to serve the dashboard on")
    flag.Parse()

    dash := dashboard.New(*orkestraURL)

    http.Handle("/dashboard/", http.StripPrefix("/dashboard", dash))
    http.Handle("/", http.RedirectHandler("/dashboard", http.StatusMovedPermanently))

    log.Printf("Dashboard starting on :%s", *port)
    log.Fatal(http.ListenAndServe(":"+*port, nil))
}