package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/adfjadfj16-a11y/Afarias19/internal/app"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dataFile := os.Getenv("DATA_FILE")
	if dataFile == "" {
		dataFile = "data/leads.json"
	}

	server, err := app.NewServer(app.Config{
		DataFile:   dataFile,
		AdminToken: os.Getenv("ADMIN_TOKEN"),
	})
	if err != nil {
		log.Fatal(err)
	}
	if os.Getenv("ADMIN_TOKEN") == "" {
		log.Print("ADMIN_TOKEN no está configurado; /api/admin/leads permanecerá inaccesible hasta definirlo")
	}

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("Afarias19 SaaS MVP escuchando en http://localhost:%s", port)
	log.Fatal(httpServer.ListenAndServe())
}
