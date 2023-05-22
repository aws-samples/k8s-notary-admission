package workloads

import (
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/admission/v1"
	a1 "k8s.io/api/apps/v1"
	b1 "k8s.io/api/batch/v1"
	pv1 "k8s.io/api/core/v1"
	"notary-admission/pkg/admissioncontroller"
	"notary-admission/pkg/admissioncontroller/verifier"
	log "notary-admission/pkg/logging"
	"notary-admission/pkg/notation"
)

// Result contains the result of an admission request
type Result struct {
	Allowed bool
	Msg     string
}

// Workload contains the workload processing data
type Workload struct {
	Kind      string
	Name      string
	Namespace string
	Images    []string
	Error     error
}

// NewValidationHook creates a new instance of pods validation hook
func NewValidationHook() admissioncontroller.Hook {
	return admissioncontroller.Hook{
		Create: validate(),
		Update: validate(),
	}
}

// parsePod parses the pod object from kind in the request
func parse(object []byte) *Workload {
	wl := Workload{}
	var result map[string]interface{}
	err := json.Unmarshal(object, &result)
	if err != nil {
		wl.Error = err
		return &wl
	}

	log.Log.Debugf("result=%+v", result)

	kind := result["kind"]
	wl.Kind = kind.(string)
	var spec pv1.PodSpec

	switch wl.Kind {
	case "Deployment":
		var d a1.Deployment
		if err = json.Unmarshal(object, &d); err != nil {
			wl.Error = err
			return &wl
		}
		wl.Name = d.Name
		wl.Namespace = d.Namespace
		spec = d.Spec.Template.Spec
	case "Pod":
		var p pv1.Pod
		if err = json.Unmarshal(object, &p); err != nil {
			wl.Error = err
			return &wl
		}
		wl.Name = p.Name
		wl.Namespace = p.Namespace
		spec = p.Spec
	case "ReplicaSet":
		var r a1.ReplicaSet
		if err = json.Unmarshal(object, &r); err != nil {
			wl.Error = err
			return &wl
		}
		wl.Name = r.Name
		wl.Namespace = r.Namespace
		spec = r.Spec.Template.Spec
	case "DaemonSet":
		var d a1.DaemonSet
		if err = json.Unmarshal(object, &d); err != nil {
			wl.Error = err
			return &wl
		}
		wl.Name = d.Name
		wl.Namespace = d.Namespace
		spec = d.Spec.Template.Spec
	case "CronJob":
		var c b1.CronJob
		if err = json.Unmarshal(object, &c); err != nil {
			wl.Error = err
			return &wl
		}
		wl.Name = c.Name
		wl.Namespace = c.Namespace
		spec = c.Spec.JobTemplate.Spec.Template.Spec
	case "Job":
		var j b1.Job
		if err = json.Unmarshal(object, &j); err != nil {
			wl.Error = err
			return &wl
		}
		wl.Name = j.Name
		wl.Namespace = j.Namespace
		spec = j.Spec.Template.Spec
	case "StatefulSet":
		var s a1.StatefulSet
		if err = json.Unmarshal(object, &s); err != nil {
			wl.Error = err
			return &wl
		}
		wl.Name = s.Name
		wl.Namespace = s.Namespace
		spec = s.Spec.Template.Spec
	default: // unsupported kind
		wl.Error = fmt.Errorf("kind %s not supported by validation controller", kind)
		return &wl
	}

	var images []string
	for _, c := range spec.Containers {
		images = append(images, c.Image)
	}
	for _, c := range spec.InitContainers {
		images = append(images, c.Image)
	}
	for _, c := range spec.EphemeralContainers {
		images = append(images, c.Image)
	}

	wl.Images = images

	return &wl
}

// validate validates workload operations
func validate() admissioncontroller.AdmitFunc {
	var w []string
	return func(ar *v1.AdmissionRequest) (*admissioncontroller.Result, error) {
		wl := parse(ar.Object.Raw)
		if wl.Error != nil {
			log.Log.Errorf("parse pod error: %v", wl.Error)
			return &admissioncontroller.Result{Msg: wl.Error.Error()}, nil
		}

		log.Log.Debugf("workload: %+v", wl)

		log.Log.Debugf("workload images = %v", wl.Images)
		v := verifier.GetEcrv().VerifySubjects(wl.Images)

		if v.Error != nil {
			log.Log.Errorf("verification error: %s, %v", v.Message, v.Error)
			return &admissioncontroller.Result{Msg: notation.ValidationFailed}, nil
		}

		var i []string
		for _, res := range v.Responses {
			log.Log.Debugf("notation Response for %s: %v", res.Image, res)

			if res.Error != nil {
				log.Log.Debugf("%s %s , in %s namespace, notation response error: %v",
					wl.Name, wl.Kind, wl.Namespace, res.Error)
				return &admissioncontroller.Result{Msg: fmt.Sprintf("%s image, in %s %s, in %s namespace, failed signature validation",
					res.Image, wl.Name, wl.Kind, wl.Namespace)}, nil
			}

			i = append(i, res.Image)
			w = append(w, res.Warning)
		}

		message := fmt.Sprintf("%s %s in %s namespace, images verified: %v", wl.Name, wl.Kind, wl.Namespace, i)
		log.Log.Debug(message)
		return &admissioncontroller.Result{
			Allowed:  true,
			Msg:      message,
			Warnings: w,
		}, nil
	}
}
