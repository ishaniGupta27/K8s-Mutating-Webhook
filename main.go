package main

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"os"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	entryLog = log.Log.WithName("entrypoint")
)

var webhooks = []rotator.WebhookInfo{
	{
		Name: "test-webhook-service-mutating-webhook-configuration",
		Type: rotator.Mutating,
	},
}

/*// readJSON from request body
func readJSON(r *http.Request, v interface{}) error {
	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		return fmt.Errorf("invalid JSON input")
	}

	return nil
}

func jsonOk(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, v)
}

// writeJSON to response body
func writeJSON(w http.ResponseWriter, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, fmt.Sprintf("json encoding error: %v", err), http.StatusInternalServerError)
		return
	}

	writeBytes(w, b)
}
func writeBytes(w http.ResponseWriter, b []byte) {
	_, err := w.Write(b)
	if err != nil {
		http.Error(w, fmt.Sprintf("write error: %v", err), http.StatusInternalServerError)
		return
	}
}

type testHandler struct{}

func (t *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello test..")
}*/

/*func handleMutate(w http.ResponseWriter, r *http.Request) {
	// read the body / request
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		sendError(err, w)
		return
	}

	// mutate the request
	mutated, err := Mutate(body, true)
	if err != nil {
		sendError(err, w)
		return
	}

	// and write it back
	w.WriteHeader(http.StatusOK)
	w.Write(mutated)
}

func sendError(err error, w http.ResponseWriter) {
	log.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "%s", err)
}*/

func main() {

	entryLog.Info("Setting certs ...3.3...again")
	logger := NewLogger()
	logger.AddFlags()
	log.SetLogger(logger.Get())

	webhookCertDir := "/certs"
	disableCertRotation := false
	secretNamespace := "test-webhook-service-system"
	secretName := "test-webhook-service-server-cert"
	caName := "test-webhook-service-ca"
	caOrganization := "test-webhook-service"
	serviceName := "test-webhook-service-service"
	serviceNamespace := "test-webhook-service-system"

	// DNSName is <service name>.<namespace>.svc
	dnsName := fmt.Sprintf("%s.%s.svc", serviceName, serviceNamespace)
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	config := ctrl.GetConfigOrDie()
	//config.UserAgent = version.GetUserAgent("webhook")

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:             scheme,
		LeaderElection:     false,
		MetricsBindAddress: "0", //nothing
		CertDir:            webhookCertDir,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(c)
		},
	})
	if err != nil {
		entryLog.Info("Failure unable to set up controller manager")
		entryLog.Info(err.Error())
		os.Exit(1)
	}

	// Make sure certs are generated and valid if cert rotation is enabled.
	setupFinished := make(chan struct{})
	if !disableCertRotation {
		entryLog.Info("setting up cert rotation..3.3")
		if err := rotator.AddRotator(mgr, &rotator.CertRotator{
			SecretKey: types.NamespacedName{
				Namespace: secretNamespace,
				Name:      secretName,
			},
			CertDir:        webhookCertDir,
			CAName:         caName,
			CAOrganization: caOrganization,
			DNSName:        dnsName,
			IsReady:        setupFinished,
			Webhooks:       webhooks,
		}); err != nil {
			entryLog.Info("Failure unable to set up cert rotation")
			entryLog.Info(err.Error())
			os.Exit(1)
		}
	} else {
		close(setupFinished)
	}

	go setupWebhook(mgr, setupFinished)

	entryLog.Info("starting manager..3.3")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Info("Failure unable to run manager")
		entryLog.Info(err.Error())
		os.Exit(1)
	}

}

func setupWebhook(mgr manager.Manager, setupFinished chan struct{}) {
	// Block until the setup (certificate generation) finishes.
	<-setupFinished

	entryLog.Info("Starting server ...3.3")

	// setup webhooks
	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()
	//hookServer.TLSMinVersion = "1.3"

	entryLog.Info("registering webhook to the webhook server")
	podMutator, err := NewPodMutator(mgr.GetClient(), mgr.GetAPIReader())
	if err != nil {
		entryLog.Info("FAILURE>>>>>")
		entryLog.Info(err.Error())
	}

	hookServer.Register("/test", &webhook.Admission{Handler: podMutator})

	/*
		mux := http.NewServeMux()

		mux.HandleFunc("/test", handleRoot)
		//mux.HandleFunc("/mutate", handleMutate)

		s := &http.Server{
			Addr:           ":8443",
			Handler:        mux,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1048576
		}

		if err := s.ListenAndServeTLS("/certs/tls.crt", "/certs/tls.key"); err != nil {
			entryLog.Info("Failure unable to ListenAndServeTLS")
			entryLog.Info(err.Error())
			os.Exit(1)
		}*/
}

// podMutator mutates pod objects to add project service account token volume
type podMutator struct {
	client client.Client
	// reader is an instance of mgr.GetAPIReader that is configured to use the API server.
	// This should be used sparingly and only when the client does not fit the use case.
	reader  client.Reader
	decoder *admission.Decoder
}

// NewPodMutator returns a pod mutation handler
func NewPodMutator(client client.Client, reader client.Reader) (admission.Handler, error) {

	return &podMutator{
		client: client,
		reader: reader,
	}, nil
}

// PodMutator adds projected service account volume for incoming pods if service account is annotated
func (m *podMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	entryLog.Info("We are here inside Handle..3.3")

	// unmarshal the pod from the AdmissionRequest

	pod := &corev1.Pod{}
	err := m.decoder.Decode(req, pod)
	if err == nil {
		entryLog.Info("We have a POD req. Let's mutate it !!!")
		return PodHandler(pod, req)
	} else {
		entryLog.Info("Not Pod")
	}

	pvc := &corev1.PersistentVolumeClaim{}
	err = m.decoder.Decode(req, pvc)
	if err == nil {
		entryLog.Info("We have a persistentvolumeclaims req. Let's mutate it !!!")
		return PvcHandler(pvc, req)
	} else {
		entryLog.Info("Not PVC")
	}

	return admission.Errored(http.StatusBadRequest, err)

}

func PodHandler(pod *corev1.Pod, req admission.Request) admission.Response {
	entryLog.Info("We are inside PodHandler !!!...2")
	newPod := pod.DeepCopy()
	ann := newPod.ObjectMeta.Annotations
	ann["test"] = "ishani"

	entryLog.Info("We are here !!!...3")
	marshaledPod, err := json.Marshal(newPod)
	if err != nil {
		entryLog.Info("failed to marshal pod object")
		entryLog.Info(err.Error())
		return admission.Errored(http.StatusBadRequest, err)
	}

	entryLog.Info("We are done adding annotation..")
	entryLog.Info(string(marshaledPod))

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func PvcHandler(pvc *corev1.PersistentVolumeClaim, req admission.Request) admission.Response {
	entryLog.Info("We are inside PvcHandler !!!")
	newPvc := pvc.DeepCopy()
	//newVal := resource.Quantity{}
	//newVal.Set(util.GiBToBytes(100))
	newPvc.Spec.Resources.Requests["storage"] = resource.MustParse("100Gi")

	entryLog.Info("We are done creating modified pvc.")
	marshaledPvc, err := json.Marshal(newPvc)
	if err != nil {
		entryLog.Info("failed to marshal pod object")
		entryLog.Info(err.Error())
		return admission.Errored(http.StatusBadRequest, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPvc)
}

// InjectDecoder injects the decoder
func (m *podMutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

func getResourceList(storage string) corev1.ResourceList {
	res := corev1.ResourceList{}
	if storage != "" {
		res[corev1.ResourceStorage] = resource.MustParse(storage)
	}
	return res
}
