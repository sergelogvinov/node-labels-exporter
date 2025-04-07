/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Node labels Controller
package main

import (
	"context"
	goflag "flag"
	"os"
	"time"

	flag "github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"

	"github.com/sergelogvinov/node-labels-exporter/pkg/nodelabelcontroller"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	version string
	commit  string

	showVersion = flag.Bool("version", false, "Print the version and exit.")

	master       = flag.String("master", "", "Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.")
	kubeconfig   = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
	kubeAPIQPS   = flag.Float32("kube-api-qps", 5, "QPS to use while communicating with the kubernetes apiserver. Defaults to 5.0.")
	kubeAPIBurst = flag.Int("kube-api-burst", 10, "Burst to use while communicating with the kubernetes apiserver. Defaults to 10.")

	certDir = flag.String("cert-dir", "certs", "webhook certificate directory")
	port    = flag.Int("port", 9443, "The port to which the admission webhook endpoint should bind")

	metricsEndpoint = flag.String("metrics-endpoint", ":8080", "The TCP network address where the HTTPS server for diagnostics, including pprof, metrics will listen (example: `:8080`).")

	scheme = runtime.NewScheme()
)

const (
	// ResyncPeriodOfNodeInformer is the resync period of the informer for the Node objects
	ResyncPeriodOfNodeInformer = 1 * time.Hour
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
}

func main() {
	var config *rest.Config
	var err error

	opts := zap.Options{
		Development:     false,
		Level:           zapcore.InfoLevel,
		StacktraceLevel: zapcore.PanicLevel,
	}
	opts.BindFlags(goflag.CommandLine)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Set("logtostderr", "true") //nolint: errcheck
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	log := ctrl.Log.WithName("init")

	log.Info("Node Labels exporter version", "version", version, "gitCommit", commit)

	if *showVersion {
		os.Exit(0)
	}

	// get the KUBECONFIG from env if specified (useful for local/debug cluster)
	kubeconfigEnv := os.Getenv("KUBECONFIG")

	if kubeconfigEnv != "" {
		log.Info("Found KUBECONFIG environment variable set, using that..")
		kubeconfig = &kubeconfigEnv
	}

	if *master != "" || *kubeconfig != "" {
		log.Info("Either master or kubeconfig specified. building kube config from that..")
		config, err = clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	} else {
		log.Info("Building kube configs for running in cluster...")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Error(err, "Failed to create config: %v")
		os.Exit(1)
	}

	config.QPS = *kubeAPIQPS
	config.Burst = *kubeAPIBurst

	coreConfig := rest.CopyConfig(config)
	coreConfig.ContentType = runtime.ContentTypeProtobuf
	clientset, err := kubernetes.NewForConfig(coreConfig)
	if err != nil {
		log.Error(err, "Failed to create a Clientset")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    *port,
			CertDir: *certDir,
		}),
		Metrics: metricsserver.Options{
			BindAddress: *metricsEndpoint,
		},
		LeaderElection: false,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctrl.SetLogger(log)

	factory := informers.NewSharedInformerFactory(clientset, ResyncPeriodOfNodeInformer)
	nodeLister := factory.Core().V1().Nodes().Lister()

	log.Info("Starting Node Labels exporter")

	m := nodelabelcontroller.NewNodeLabelsEnvInjector(clientset, scheme, nodeLister, ctrl.Log.WithName("controllers").WithName("NodeLabelsEnvInjector"))

	mgr.GetWebhookServer().Register("/webhook", &webhook.Admission{
		Handler: admission.HandlerFunc(m.Handle),
	})

	ctx, cancel := context.WithCancel(context.Background())

	run := func(ctx context.Context) {
		defer cancel()

		factory.Start(ctx.Done())
		cacheSyncResult := factory.WaitForCacheSync(ctx.Done())
		for _, v := range cacheSyncResult {
			if !v {
				log.Info("Failed to sync Informers!")
				os.Exit(1)
			}
		}

		if err := mgr.Start(ctx); err != nil {
			log.Error(err, "problem running manage")
			os.Exit(1)
		}
	}

	go func() {
		<-ctrl.SetupSignalHandler().Done()
		cancel()
	}()

	run(ctx)
}
