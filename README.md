goocr - GO Ocr service
========================

This Golang based project provides a microservice that offers a REST API and a Web
view to convert PDF's and Images to Text, using Tesseract OCR scanner.

Just a proof-of-concept at this point. For future development it will be split in a
multi-tier application architecture for better escalability - again for instructional
purposes.

----

### 1. How to build and run:

```
docker compose up --build
```

### 2. Main Web Views

The service provides some minimalistic webviews to use the functionalities.
```
http://localhost:8080/web/pdf
http://localhost:8080/web/img
```

## 3. Endpoints

### 3.1 API Endpoints for PDF submission
```
http://localhost:8080/api/upload/pdf
```
### 3.2 API endpoint for Image submission
```
http://localhost:8080/api/upload/img
```

## 4. Frameworks
This projects uses the following SDK's:
- [Tesseract OCR](http://github.com/tesseract-ocr) : OCR Engine
- [GhostScript](http://www.ghostscript.com): PDF interpreter used to convert PDF to a set of images (per page)

(C) 2024 Simone Chiorazzo

