package main

import (
	"flag"
	"os"
	"strconv"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	crdv1 "github.com/flexlet/flexlb-kube-controller/api/v1"
	"github.com/flexlet/flexlb-kube-controller/controllers"
	"github.com/flexlet/flexlb-kube-controller/handlers"
	//+kubebuilder:scaffold:imports
)

const (
	defaultRefreshInterval    = 30
	defaultErrorRetryInterval = 1
	defaultNamespace          = "kube-system"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(crdv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var (
		// TODO: auto generated
		metricsAddr          = flag.String("metrics-bind-address", os.Getenv("METRICS_BIND_ADDRESS"), "The address the metric endpoint binds to.")
		probeAddr            = flag.String("health-probe-bind-address", os.Getenv("HEALTH_PROBE_BIND_ADDRESS"), "The address the probe endpoint binds to.")
		enableLeaderElection = flag.Bool("leader-elect", false, "Enable leader election for controller manager.")

		tlsCaCert     = flag.String("tls-ca-cert", os.Getenv("FLEXLB_TLS_CA_CERT"), "FlexLB API server TLS ca certificate")
		tlsClientCert = flag.String("tls-client-cert", os.Getenv("FLEXLB_TLS_CLIENT_CERT"), "FlexLB API server TLS client certificate")
		tlsClientKey  = flag.String("tls-client-key", os.Getenv("FLEXLB_TLS_CLIENT_KEY"), "FlexLB API server TLS client key")
		tlsInsecure   = flag.Bool("tls-insecure", true, "FlexLB API server ignore insecure server certificate")

		refreshInterval = flag.String("refresh-interval", os.Getenv("FLEXLB_REFRESH_INTERVAL"), "Instance refresh interval in seconds")
		namespace       = flag.String("namespace", os.Getenv("FLEXLB_NAMESPACE"), "Namespace for flexlb clusters and temporary pods")
		probePodImage   = flag.String("probe-pod-image", os.Getenv("FLEXLB_PROBE_POD_IMAGE"), "Node probe pod image")
	)

	// zap command line options:
	//   --zap-devel      enalbe development mode：encoder = consoleEncoder，logLevel = Debug，stackTraceLevel = Warn
	//                    default is production mode: encoder = jsonEncoder，logLevel = Info，stackTraceLevel = Error
	//   --zap-encoder           json, console
	//   --zap-log-level         debug, info, error
	//   --zap-stacktrace-level  info,error

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     *metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: *probeAddr,
		LeaderElection:         *enableLeaderElection,
		LeaderElectionID:       "82b77363.flexlb.flexlet.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// setup handler
	handler := handlers.NewHandler(*tlsCaCert, *tlsClientCert, *tlsClientKey, *tlsInsecure, *namespace, *probePodImage, mgr.GetEventRecorderFor("flexlb-handler"))

	refreshSeconds, err := strconv.Atoi(*refreshInterval)
	if err != nil {
		refreshSeconds = defaultRefreshInterval
	}

	if namespace == nil || len(*namespace) == 0 {
		ns := defaultNamespace
		namespace = &ns
	}

	if err = (&controllers.FlexLBClusterReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Namespace:     *namespace,
		ChangeHandler: handler.ClusterChanged,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FlexLBCluster")
		os.Exit(1)
	}

	if err = (&controllers.FlexLBInstanceReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		RefreshInterval: time.Duration(refreshSeconds) * time.Second,
		ChangeHandler:   handler.InstanceChanged,
		DeleteHandler:   handler.InstanceDeleted,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FlexLBInstance")
		os.Exit(1)
	}

	if err = (&controllers.NodeReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		ChangeHandler: handler.NodeChanged,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Node")
		os.Exit(1)
	}

	if err = (&controllers.ServiceReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		ChangeHandler: handler.ServiceChanged,
		DeleteHandler: handler.ServiceDeleted,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Node")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
