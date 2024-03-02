#
# Tesseract Hackathon Docker Image
#

FROM ubuntu:22.04
ARG TARGETOS
ARG TARGETARCH
MAINTAINER chiora93@gmail.com

# Install essential packages needed for compilatiion / execution of Tesseract.
RUN apt-get update && apt-get install -y \
  autoconf \
  automake \
  autotools-dev \
  build-essential \
  checkinstall \
  libjpeg-dev \
  libpng-dev \
  libtiff-dev \
  libtool \
  libicu-dev \
  libpango1.0-0 \
  libpango1.0-dev \
  icu-devtools \
  python3 \
  python3-tornado \
  wget \
  zlib1g-dev \
  git \
  imagemagick \
  ghostscript \
  tesseract-ocr \
  libtesseract-dev \
  tesseract-ocr-eng \
  tesseract-ocr-fra \
  tesseract-ocr-deu \
  tesseract-ocr-ita

RUN wget -qO- https://dl.google.com/go/go1.21.6.${TARGETOS}-${TARGETARCH}.tar.gz | tar xvz -C /usr/local
ENV PATH $PATH:/usr/local/go/bin

# Set GOPATH
ENV GOPATH /go
ENV PATH /go/bin:$PATH

# Set Tesseract Training data location
ENV TESSDATA_PREFIX /usr/share/tesseract-ocr/4.00/tessdata

# Copy code to image
COPY . /go/src/github.com/chiora93/goocr

WORKDIR /go/src/github.com/chiora93/goocr

RUN go install -v -a github.com/chiora93/goocr/cmd/goocr/...

CMD /go/bin/goocr

EXPOSE 80
