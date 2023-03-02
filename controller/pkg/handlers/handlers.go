package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"notary-admission/pkg/admissioncontroller"

	v1 "k8s.io/api/admission/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	log "notary-admission/pkg/logging"
	"sync"
)

type logic struct {
	m sync.Mutex
}

type controller struct {
	l *logic
}

// admissionHandler represents the HTTP handler for an admission webhook
type admissionHandler struct {
	l       *logic
	decoder runtime.Decoder
}

// newAdmissionHandler returns an instance of AdmissionHandler
func newAdmissionHandler() *admissionHandler {
	return &admissionHandler{
		decoder: serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer(),
	}
}

// Serve returns a handlers.HandlerFunc for an admission webhook
func (h *admissionHandler) Serve(hook admissioncontroller.Hook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, ContentTypeJson)
		if r.Method != http.MethodPost {
			http.Error(w, fmt.Sprint("invalid method, only POST requests are allowed"), http.StatusMethodNotAllowed)
			return
		}

		if contentType := r.Header.Get(HeaderContentType); contentType != ContentTypeJson {
			http.Error(w, fmt.Sprint("only content type 'application/json' is supported"), http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not read request body: %v", err), http.StatusBadRequest)
			return
		}

		log.Log.Debugf("Request body: %s", string(body))

		var review v1.AdmissionReview
		if _, _, err := h.decoder.Decode(body, nil, &review); err != nil {
			http.Error(w, fmt.Sprintf("could not deserialize request: %v", err), http.StatusBadRequest)
			return
		}

		//log.Log.Debug("Admission Review: %v", review)
		if review.Request == nil {
			http.Error(w, "malformed admission review: request is nil", http.StatusBadRequest)
			return
		}
		log.Log.Debugf("Admission Review object: %v", string(review.Request.Object.Raw))

		result, err := hook.Execute(review.Request)
		if err != nil {
			log.Log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		admissionResponse := v1.AdmissionReview{
			TypeMeta: meta.TypeMeta{
				Kind:       AdmissionReviewKind,
				APIVersion: AdmissionReviewVersion,
			},
			Response: &v1.AdmissionResponse{
				UID:     review.Request.UID,
				Allowed: result.Allowed,
				Result:  &meta.Status{Message: result.Msg},
			},
		}

		log.Log.Debugf("Admission Response: %v", admissionResponse)

		res, err := json.Marshal(admissionResponse)
		if err != nil {
			log.Log.Error(err)
			http.Error(w, fmt.Sprintf("could not marshal response: %v", err), http.StatusInternalServerError)
			return
		}

		log.Log.Debugf("Response JSON: %s", string(res))

		log.Log.Infof("Webhook [%s - %s] - Allowed: %t, Message: %s", r.URL.Path,
			review.Request.Operation, result.Allowed, result.Msg)
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}

// healthz returns a handlers.HandlerFunc for a health checks
func (c *controller) healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, fmt.Sprint("invalid method, only GET or HEAD requests are allowed"), http.StatusMethodNotAllowed)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}
