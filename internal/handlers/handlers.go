package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/chiora93/goocr/internal/schema"
	"github.com/chiora93/goocr/internal/wrappers"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/nu7hatch/gouuid"
)

const (
	NumberParallelRoutines = 4
)

type Handlers struct {
	throttle  chan int
	uploadDir string
}

func NewHandlers(uploadDir string) *Handlers {
	return &Handlers{
		throttle:  make(chan int, NumberParallelRoutines),
		uploadDir: uploadDir,
	}
}

func (h *Handlers) GuiUploadPDF(w http.ResponseWriter, _ *http.Request) {
	log.Info("Request to handlers image service via GUI")

	microPage := `
		<html>
			<title>GOOCR - OCR microservice demo</title>
			<script src="//ajax.googleapis.com/ajax/libs/jquery/1.10.2/jquery.min.js"></script>
			<body>
				<h2>GOOCR - OCR microservice demo</h2>
				<h4>PDF File Submission</h4>
				</pre>	
				<form action="/api/v1/documents/pdf/ocr-scan" method="post" enctype="multipart/form-data">
					<input type="file" name="the_file" />
					<input type="submit" value="Submit PDF" />
				</form>
				<pre class="prettyprint">
				<div id="result"></div>
			</body>
		</html>`

	_, _ = fmt.Fprintf(w, microPage)
}

func (h *Handlers) GuiUploadImage(w http.ResponseWriter, _ *http.Request) {
	log.Info("Request to handlers image service via GUI")

	microPage := `
		<html>
			<title>GOOCR - OCR microservice demo</title>
			<script src="//ajax.googleapis.com/ajax/libs/jquery/1.10.2/jquery.min.js"></script>
			<body>
				<h2>GOOCR - OCR microservice demo</h2>
				<h4>JPG Image File Submission</h4>
				</pre>
				<form action="/api/v1/documents/img/ocr-scan" method="post" enctype="multipart/form-data">
					<input type="file" name="the_file" />
					<input type="submit" value="Submit JPG" />
				</form>
				<pre class="prettyprint">
				<div id="result"></div>
			</body>
		</html>`

	_, _ = fmt.Fprintf(w, microPage)
}

func (h *Handlers) ScanImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	log.Info("Request to handlers image service")

	var (
		err        error
		submission schema.SubmissionDetails
	)

	if !h.validateInput(w, r, &submission) {
		log.WithField("submissions", submission).Error("Invalid submission")
		http.Error(w, "Unable to process request", http.StatusBadRequest)
		return
	}

	var tempPath string
	var numberOfPages int
	var txtsOutputPath string

	for _, fheaders := range r.MultipartForm.File {
		for _, hdr := range fheaders {
			submission.FileName = hdr.Filename
			// open uploaded
			var infile multipart.File
			if infile, err = hdr.Open(); err != nil {
				log.WithField("imgFilename", hdr.Filename).WithError(err).Error("Error uploading image file")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}
			// open destination
			var outfile *os.File

			// Save the file into the docker container disk,
			generatedUUID, err := uuid.NewV4()
			if err != nil {
				log.WithError(err).Error("Error creating UUID")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}

			submission.UUID = generatedUUID.String()

			tempPath = path.Join(h.uploadDir, generatedUUID.String())
			log.WithFields(log.Fields{
				"tmpDir":   tempPath,
				"fileName": hdr.Filename,
			}).Info("Storing submitted Image")

			if err := os.MkdirAll(tempPath, os.ModePerm); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"tempPath": tempPath,
				}).Error("Unable to write temporary folder")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}

			outfile, err = os.Create(filepath.Join(tempPath, schema.DocumentImageName))
			if err != nil {
				log.WithError(err).Error("Error creating temporary handlers file")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}
			defer func(outfile *os.File) {
				err := outfile.Close()
				if err != nil {
					log.WithError(err).Error("Failed to close file")
				}
			}(outfile)

			// 32K buffer copy
			if _, err = io.Copy(outfile, infile); err != nil {
				log.WithError(err).Error("Error while copying file")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}

			log.WithField("MaxConcurrency", NumberParallelRoutines).Info("Launching main Tesseract text extraction worker")
			txtsOutputPath = path.Join(tempPath, schema.TextFolderName)
			if err := os.MkdirAll(txtsOutputPath, os.ModePerm); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"txtsOutputPath": txtsOutputPath,
				}).Error("Unable to write text output folder")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}

			var wg sync.WaitGroup
			numberOfPages = h.processParalellOCR(tempPath, "jpg", txtsOutputPath, &wg)
		}
	}

	submission.NumberOfPages = numberOfPages
	submission.Pages = h.generatePageDetails(txtsOutputPath)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(submission); err != nil {
		log.WithError(err).Error("Error marshalling submission JSON")
	}
}

func (h *Handlers) ScanPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	log.Info("Request to handlers pdf service")

	var (
		err        error
		submission schema.SubmissionDetails
	)

	if !h.validateInput(w, r, &submission) {
		log.WithField("submissions", submission).Error("Invalid submission")
		http.Error(w, "Unable to process request", http.StatusBadRequest)
		return
	}

	var tempPath string
	var numberOfPages int
	var txtsOutputPath string

	for _, fheaders := range r.MultipartForm.File {
		for _, hdr := range fheaders {
			submission.FileName = hdr.Filename
			// open uploaded
			var infile multipart.File
			if infile, err = hdr.Open(); nil != err {
				log.WithField("PDFFilename", hdr.Filename).WithError(err).Error("Error uploading PDF file")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}
			// open destination
			var outfile *os.File

			// Save the file into the docker container disk,
			generatedUUID, err := uuid.NewV4()
			if err != nil {
				log.WithError(err).Error("Error creating UUID")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}

			submission.UUID = generatedUUID.String()

			tempPath = path.Join(h.uploadDir, generatedUUID.String())
			log.WithFields(log.Fields{
				"tmpDir":   tempPath,
				"fileName": hdr.Filename,
			}).Info("Storing submitted PDF")

			if err := os.MkdirAll(tempPath, os.ModePerm); err != nil {
				log.WithError(err).Error("Error creating temporary directory")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}

			if outfile, err = os.Create(filepath.Join(tempPath, schema.DocumentFileName)); nil != err {
				log.WithError(err).Error("Error creating temporary handlers file")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}
			defer func(outfile *os.File) {
				err := outfile.Close()
				if err != nil {
					log.WithError(err).Error("Error closing file")
				}
			}(outfile)

			// 32K buffer copy
			if _, err = io.Copy(outfile, infile); nil != err {
				log.WithError(err).Error("Error while copying file")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}

			// Generates Images from the PDF
			imagesOutputPath := path.Join(tempPath, schema.ImagesFolderName)
			if err := os.MkdirAll(imagesOutputPath, os.ModePerm); err != nil {
				log.WithError(err).Error("Error creating images output directory")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}

			pdfFilePath := path.Join(tempPath, schema.DocumentFileName)
			if err := wrappers.ExtractPdfToImagesFromPDF(pdfFilePath, imagesOutputPath); err != nil {
				log.WithField("pdfFilePath", pdfFilePath).WithError(err).Error("Unable to extract images from PDF")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}

			var wg sync.WaitGroup
			log.WithField("MaxConcurrency", NumberParallelRoutines).Info("Launching main Tesseract text extraction worker")
			txtsOutputPath = path.Join(tempPath, schema.TextFolderName)
			if err := os.MkdirAll(txtsOutputPath, os.ModePerm); err != nil {
				log.WithError(err).Error("Error creating texts output directory")
				http.Error(w, "Unable to process request", http.StatusInternalServerError)
				return
			}

			numberOfPages = h.processParalellOCR(imagesOutputPath, "jpg", txtsOutputPath, &wg)

		}
	}
	submission.NumberOfPages = numberOfPages
	submission.Pages = h.generatePageDetails(txtsOutputPath)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(submission); err != nil {
		log.WithError(err).Error("Error marshalling submission JSON")
	}
}

func (h *Handlers) processParalellOCR(imagesDirectoryPath string, imageExtension string, textOutPutDirectory string, wg *sync.WaitGroup) int {
	imageFilesList, _ := os.ReadDir(imagesDirectoryPath)

	numberFiles := 0

	for _, f := range imageFilesList {
		if !strings.HasSuffix(f.Name(), imageExtension) || f.IsDir() {
			continue
		}
		imagePath := path.Join(imagesDirectoryPath, f.Name())
		h.throttle <- 1 // whatever number
		wg.Add(1)
		log.WithFields(log.Fields{
			"imagesDirectoryPath": imagesDirectoryPath,
			"imageExtension":      imageExtension,
			"textOutPutDirectory": textOutPutDirectory,
		})
		go wrappers.ExtractPlainTextFromImage(imagePath, "ita", textOutPutDirectory, f.Name(), wg, h.throttle)

		numberFiles++
	}
	wg.Wait()

	return numberFiles
}

func (h *Handlers) generatePageDetails(textsDirectory string) []schema.PageDetails {
	txtsFilesList, _ := os.ReadDir(textsDirectory)

	pages := make([]schema.PageDetails, len(txtsFilesList))

	pageNumber := 0

	for _, txt := range txtsFilesList {
		txtPath := path.Join(textsDirectory, txt.Name())
		data, err := os.ReadFile(txtPath)

		if err != nil {
			log.WithError(err).Error("Cannot read txt file")
		}

		page := schema.PageDetails{
			PageNumber: pageNumber + 1,
			Text:       string(data),
		}
		pages[pageNumber] = page
		pageNumber++
	}

	return pages
}

func (h *Handlers) validateInput(_ http.ResponseWriter, req *http.Request, _ *schema.SubmissionDetails) bool {
	// Need to call ParseMultipartForm first, so we can check if a file is contained
	// parameter for max memory taken from https://golang.org/src/net/http/request.go
	// Note that this is 32mb, and is probably why 40mb files are failing
	_ = req.ParseMultipartForm(32 << 20)

	if req.MultipartForm == nil || len(req.MultipartForm.File) == 0 {
		log.Error("No file passed in")
		return false
	}

	var maxSizeBits int64
	maxSizeBits = (1 << 20) * schema.MaxSizeMB

	if err := req.ParseMultipartForm(maxSizeBits); nil != err {
		log.Error("File exceeds maximum size")
		return false
	}

	return true
}
