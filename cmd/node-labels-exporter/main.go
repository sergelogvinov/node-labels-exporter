/*
Copyright 2024 The Kubernetes Authors.

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
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog/v2"
)

var (
	version string
	commit  string

	showVersion = flag.Bool("version", false, "Print the version and exit.")

	master       = flag.String("master", "", "Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.")
	kubeconfig   = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
	kubeAPIQPS   = flag.Float32("kube-api-qps", 5, "QPS to use while communicating with the kubernetes apiserver. Defaults to 5.0.")
	kubeAPIBurst = flag.Int("kube-api-burst", 10, "Burst to use while communicating with the kubernetes apiserver. Defaults to 10.")

	httpEndpoint = flag.String("http-endpoint", ":8443", "The TCP network address where the HTTPS server for diagnostics, including pprof, metrics will listen (example: `:8443`).")
	metricsPath  = flag.String("metrics-path", "/metrics", "The HTTP path where prometheus metrics will be exposed. Default is `/metrics`.")
)

const (
	// ResyncPeriodOfNodeInformer is the resync period of the informer for the Node objects
	ResyncPeriodOfNodeInformer = 1 * time.Hour
)

func main() {
	var config *rest.Config
	var err error

	klog.InitFlags(nil)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Set("logtostderr", "true") //nolint: errcheck
	flag.Parse()

	klog.V(2).InfoS("Version", "version", "0.1", "gitVersion", version, "gitCommit", commit)

	if *showVersion {
		klog.Infof("Node labels Controller: version %v, GitVersion %s", "0.1", version)
		os.Exit(0)
	}

	ctx := context.Background()

	// get the KUBECONFIG from env if specified (useful for local/debug cluster)
	kubeconfigEnv := os.Getenv("KUBECONFIG")

	if kubeconfigEnv != "" {
		klog.Infof("Found KUBECONFIG environment variable set, using that..")
		kubeconfig = &kubeconfigEnv
	}

	if *master != "" || *kubeconfig != "" {
		klog.Infof("Either master or kubeconfig specified. building kube config from that..")
		config, err = clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	} else {
		klog.Infof("Building kube configs for running in cluster...")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Fatalf("Failed to create config: %v", err)
	}

	config.QPS = *kubeAPIQPS
	config.Burst = *kubeAPIBurst

	coreConfig := rest.CopyConfig(config)
	coreConfig.ContentType = runtime.ContentTypeProtobuf
	clientset, err := kubernetes.NewForConfig(coreConfig)
	if err != nil {
		klog.ErrorS(err, "Failed to create a Clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	factory := informers.NewSharedInformerFactory(clientset, ResyncPeriodOfNodeInformer)
	// nodeLister := factory.Core().V1().Nodes().Lister()

	// Prepare http endpoint for metrics
	mux := http.NewServeMux()
	gatherers := prometheus.Gatherers{
		legacyregistry.DefaultGatherer,
	}

	if *httpEndpoint != "" {
		// m := libmetrics.New("controller")
		reg := prometheus.NewRegistry()
		reg.MustRegister([]prometheus.Collector{}...)
		// provisionerOptions = append(provisionerOptions, controller.MetricsInstance(m))
		gatherers = append(gatherers, reg)

		mux.Handle(*metricsPath,
			promhttp.InstrumentMetricHandler(
				reg,
				promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})))

		go func() {
			klog.Infof("ServeMux listening at %q", *httpEndpoint)
			err := http.ListenAndServe(*httpEndpoint, mux)
			if err != nil {
				klog.Fatalf("Failed to start HTTP server at specified address (%q) and metrics path (%q): %s", *httpEndpoint, *metricsPath, err)
			}
		}()
	}

	klog.InfoS("Starting node labels exporter")

	run := func(ctx context.Context) {
		factory.Start(ctx.Done())
		cacheSyncResult := factory.WaitForCacheSync(ctx.Done())
		for _, v := range cacheSyncResult {
			if !v {
				klog.Fatalf("Failed to sync Informers!")
			}
		}

		// nodeLabelsController.Run(ctx)
	}

	run(ctx)
}
