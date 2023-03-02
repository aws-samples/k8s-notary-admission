package pods

import (
	"encoding/json"

	"notary-admission/pkg/admissioncontroller"

	"fmt"
	v1 "k8s.io/api/admission/v1"
	pv1 "k8s.io/api/core/v1"
	"notary-admission/pkg/admissioncontroller/verifier"
	log "notary-admission/pkg/logging"
	"notary-admission/pkg/notation"
)

// Result contains the result of an admission request
type Result struct {
	Allowed bool
	Msg     string
}

// NewValidationHook creates a new instance of pods validation hook
func NewValidationHook() admissioncontroller.Hook {
	return admissioncontroller.Hook{
		Create: validatePod(),
		Update: validatePod(),
	}
}

// parsePod parses the pod object from request
func parsePod(object []byte) (*pv1.Pod, error) {
	var pod pv1.Pod
	if err := json.Unmarshal(object, &pod); err != nil {
		return nil, err
	}

	return &pod, nil
}

// ValidatePod validates pod operations
func validatePod() admissioncontroller.AdmitFunc {
	return func(ar *v1.AdmissionRequest) (*admissioncontroller.Result, error) {
		pod, err := parsePod(ar.Object.Raw)
		if err != nil {
			log.Log.Errorf("parse pod error: %v", err)
			return &admissioncontroller.Result{Msg: err.Error()}, nil
		}

		log.Log.Debugf("Pod Spec: %v", pod.Spec)

		//subjects := verifier.Subjects{}
		var images []string
		for _, c := range pod.Spec.Containers {
			images = append(images, c.Image)
		}

		//subjects.Images = images

		//v := verifier.GetEcrv().VerifySubjects(verifier.Subjects{
		//	Images: images,
		//})

		log.Log.Debugf("Pod images = %v", images)
		v := verifier.GetEcrv().VerifySubjects(images)

		if v.Error != nil {
			log.Log.Errorf("verification error: %s, %v", v.Message, v.Error)
			return &admissioncontroller.Result{Msg: notation.ValidationFailed}, nil
		}

		var i []string
		for _, res := range v.Responses {
			log.Log.Debugf("Notation Response for %s: %v", res.Image, res)

			if res.Error != nil {
				log.Log.Debugf("%s pod, in %s namespace, notation response error: %v",
					pod.Name, pod.Namespace, res.Error)
				return &admissioncontroller.Result{Msg: fmt.Sprintf("%s image, in %s pod, in %s namespace, failed signature validation",
					res.Image, pod.Name, pod.Namespace)}, nil
			}

			i = append(i, res.Image)
		}

		message := fmt.Sprintf("%s pod in %s namespace, images verified: %v", pod.Name, pod.Namespace, i)
		log.Log.Debug(message)
		return &admissioncontroller.Result{
			Allowed: true,
			Msg:     message,
		}, nil
	}
}
