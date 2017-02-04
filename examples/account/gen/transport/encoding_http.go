package transport

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"io"
	"mime"
	"net/http"

	"github.com/rightscale/aes/logger"

	"goa.design/goa.v2"
	"goa.design/goa.v2/rest"
)

// NewHTTPDecoder returns a HTTP request body decoder.
// The decoder handles the following content types:
//
// * application/json using package encoding/json
// * application/xml using package encoding/xml
// * application/gob using package encoding/gob
func NewHTTPDecoder(r *http.Request) rest.Decoder {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		// Default to JSON
		contentType = "application/json"
	} else {
		if mediaType, _, err := mime.ParseMediaType(contentType); err == nil {
			contentType = mediaType
		}
	}
	switch contentType {
	case "application/json":
		return json.NewDecoder(r.Body)
	case "application/gob":
		return gob.NewDecoder(r.Body)
	case "application/xml":
		return xml.NewDecoder(r.Body)
	default:
		return json.NewDecoder(r.Body)
	}
}

// NewHTTPEncoder returns a HTTP response encoder.
// The encoder handles the following content types:
//
// * application/json using package encoding/json
// * application/xml using package encoding/xml
// * application/gob using package encoding/gob
func NewHTTPEncoder(w http.ResponseWriter, r *http.Request) rest.Encoder {
	accept := r.Header.Get("Accept")
	if accept == "" {
		// Default to JSON
		accept = "application/json"
	} else {
		if mediaType, _, err := mime.ParseMediaType(accept); err == nil {
			accept = mediaType
		}
	}
	switch accept {
	case "application/json":
		return json.NewEncoder(w)
	case "application/gob":
		return gob.NewEncoder(w)
	case "application/xml":
		return xml.NewEncoder(w)
	default:
		return json.NewEncoder(w)
	}
}

// NewErrorHTTPEncoder returns an encoder that checks whether the error is a goa
// Error and if so sets the response status code using the error status and
// encodes the corresponding ErrorResponse struct to the response body. If the
// error is not a goa.Error then it sets the response status code to 500, writes
// the error message to the response body and logs it.
func NewErrorHTTPEncoder(w http.ResponseWriter, r *http.Request, logger goa.Logger) rest.ErrorEncoder {
	return &errorEncoder{
		w:       w,
		r:       r,
		encoder: NewHTTPEncoder(w, r),
	}
}

type errorEncoder struct {
	w       http.ResponseWriter
	r       *http.Request
	encoder rest.Encoder
}

func (e *errorEncoder) Encode(handled error) {
	switch t := handled.(type) {
	case goa.Error:
		e.w.Header().Set("Content-Type", ResponseContentType(e.r))
		e.w.WriteHeader(rest.HTTPStatus(t.Status()))
		err := e.encoder.Encode(rest.NewErrorResponse(t))
		if err != nil {
			logger.Error(e.r.Context(), "encoding", err)
		}
	default:
		b := make([]byte, 6)
		io.ReadFull(rand.Reader, b)
		id := base64.RawURLEncoding.EncodeToString(b) + ": "
		e.w.Header().Set("Content-Type", "text/plain")
		e.w.WriteHeader(http.StatusInternalServerError)
		e.w.Write([]byte(id + handled.Error()))
		logger.Error(e.r.Context(), "id", id, "error", handled.Error())
	}
}

// ResponseContentType returns the value of the Content-Type header for the
// given request.
func ResponseContentType(r *http.Request) string {
	accept := r.Header.Get("Accept")
	if accept == "" {
		// Default to JSON
		return "application/json"
	}
	if mediaType, _, err := mime.ParseMediaType(accept); err == nil {
		accept = mediaType
	}
	switch accept {
	case "application/json",
		"application/gob",
		"application/xml":
		return accept
	default:
		return "application/json"
	}
}
