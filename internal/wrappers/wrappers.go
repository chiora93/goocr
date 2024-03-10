package wrappers

import (
	"bufio"
	"fmt"
	"github.com/chiora93/goocr/internal/schema"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/otiai10/gosseract"
)

// ExtractPdfToImagesFromPDF extracts Images from the PDF file and output an image per page.
func ExtractPdfToImagesFromPDF(pdfFullPath, outputDirectory string) error {
	log.WithFields(log.Fields{
		"pdfFullPath":     pdfFullPath,
		"outputDirectory": outputDirectory,
	}).Info("Extracting Images from PDF via Ghostscript")
	pwd, err := os.Getwd()
	log.Info("Starting `gs` command from working dir: ", pwd)
	cmdArgs := []string{"-dNOPAUSE", "-dBATCH", "-sDEVICE=jpeg", "-r300", "-sOutputFile=" + outputDirectory + "/p%03d.jpg", pdfFullPath}
	log.Info("Starting command: gs ", strings.Join(cmdArgs, " "))
	cmd := exec.Command("gs", cmdArgs...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.WithError(err).Error("Error creating StdoutPipe for Cmd")
		return err
	}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			log.Printf("Ghosscript output | %s\n", scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.WithError(err).Error("Error starting Cmd")
		return err
	}

	err = cmd.Wait()
	if err != nil {
		log.WithError(err).Error("Error waiting for Cmd")
		return err
	}

	return nil
}

// ExtractPlainTextFromImage given an image file, Tesseract OCR generates a plain text file with the detected text.
func ExtractPlainTextFromImage(imageFullPath, languages, outputDirectory, textFilePrefix string, wg *sync.WaitGroup, throttle chan int) {
	defer wg.Done()

	outText := gosseract.Must(gosseract.Params{
		Src:       imageFullPath,
		Languages: languages, //eng+heb
	})

	textFilePath := filepath.Join(outputDirectory, fmt.Sprintf("%s_%s", textFilePrefix, schema.TextFileName))
	outfile, err := os.Create(textFilePath)
	if err != nil {
		log.WithError(err).Error("Error creating text file")
		return
	}
	defer func(outfile *os.File) {
		err := outfile.Close()
		if err != nil {
			log.WithError(err).Error("Error closing text file")
		}
	}(outfile)

	log.WithFields(log.Fields{
		"imageFullPath":   imageFullPath,
		"outputDirectory": outputDirectory,
		"languages":       languages,
		"textFilePath":    textFilePath,
	}).Info("Processed OCR Tesseract Instance")

	sanitizedTxt := strings.Replace(outText, "\n", " ", -1)
	_, err = outfile.WriteString(sanitizedTxt)
	if err != nil {
		log.WithError(err).Error("Error writing on output file")
		return
	}

	<-throttle
}
