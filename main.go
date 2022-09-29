package main

import (
	"fmt"
	"html"

	"net/http"
	"os"
	"time"

	//lg "github.com/ishaniGupta27/K8s-Mutating-Webhook/main"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
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

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello test.. %q", html.EscapeString(r.URL.Path))
}

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

	entryLog.Info("Setting certs ...12")
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
		MetricsBindAddress: "0",
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
		entryLog.Info("setting up cert rotation..12")
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

	entryLog.Info("starting manager..12")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Info("Failure unable to run manager")
		entryLog.Info(err.Error())
		os.Exit(1)
	}

}

func setupWebhook(mgr manager.Manager, setupFinished chan struct{}) {
	// Block until the setup (certificate generation) finishes.
	<-setupFinished

	entryLog.Info("Starting server ...11")

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
	}
}
