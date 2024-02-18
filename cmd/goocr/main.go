package main

import (
	"github.com/chiora93/goocr/internal/handlers"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/rs/cors"
)

func init() {
	log := logrus.New()
	log.Formatter = new(logrus.JSONFormatter) // default
	log.Level = logrus.DebugLevel
}

func main() {
	logrus.Info("GOOCR - Tesseract Rest Service Starting...")

	h := handlers.NewHandlers(os.Getenv("UPLOADED_FILES_DIR"))

	router := http.NewServeMux()
	router.HandleFunc("/api/v1/documents/pdf/ocr-scan", h.ScanPDF)
	router.HandleFunc("/api/v1/documents/img/ocr-scan", h.ScanImage)
	router.HandleFunc("/web/pdf", h.GuiUploadPDF)
	router.HandleFunc("/web/img", h.GuiUploadImage)

	// Wrapping the API handler in CORS default behaviors
	c := cors.New(cors.Options{
		AllowedHeaders: []string{"Origin", "Accept", "Content-Type"},
		AllowedMethods: []string{"GET", "POST"},
	})

	// Start server
	err := http.ListenAndServe(":80", c.Handler(router))
	if err != nil {
		logrus.Fatal("Error attempting to ListenAndServe: ", err)
	}
}
