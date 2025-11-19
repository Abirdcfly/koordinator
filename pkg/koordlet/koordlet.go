/*
Copyright 2022 The Koordinator Authors.

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

package agent

import (
	"fmt"
	"os"
	"time"

	topologyv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
	topologyclientset "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/generated/clientset/versioned"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientset "k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	slov1alpha1 "github.com/koordinator-sh/koordinator/apis/slo/v1alpha1"
	clientsetbeta1 "github.com/koordinator-sh/koordinator/pkg/client/clientset/versioned"
	"github.com/koordinator-sh/koordinator/pkg/client/clientset/versioned/typed/scheduling/v1alpha1"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/config"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/extension"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/metriccache"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/metrics"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/metricsadvisor"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/prediction"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/qosmanager"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/resourceexecutor"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/runtimehooks"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/statesinformer"
	statesinformerimpl "github.com/koordinator-sh/koordinator/pkg/koordlet/statesinformer/impl"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/util/system"
)

var (
	scheme = apiruntime.NewScheme()

	extensionControllerInitFuncs = map[string]extension.ControllerInitFunc{}
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = slov1alpha1.AddToScheme(scheme)
	_ = topologyv1alpha1.AddToScheme(scheme)
}

type Daemon interface {
	Run(stopCh <-chan struct{})
}

type daemon struct {
	metricAdvisor  metricsadvisor.MetricAdvisor
	statesInformer statesinformer.StatesInformer
	metricCache    metriccache.MetricCache
	qosManager     qosmanager.QOSManager
	runtimeHook    runtimehooks.RuntimeHook
	predictServer  prediction.PredictServer
	executor       resourceexecutor.ResourceUpdateExecutor

	extensionControllers []extension.Controller
}

func NewDaemon(config *config.Configuration) (Daemon, error) {
	klog.Infoln("18:34")
	// get node name
	nodeName := os.Getenv("NODE_NAME")
	if len(nodeName) == 0 {
		return nil, fmt.Errorf("failed to new daemon: NODE_NAME env is empty")
	}
	klog.Infof("NODE_NAME is %v, start time %v", nodeName, float64(time.Now().Unix()))
	metrics.RecordKoordletStartTime(nodeName, float64(time.Now().Unix()))

	system.InitSupportConfigs()
	klog.Infof("sysconf: %+v, agentMode: %v", system.Conf, system.AgentMode)
	klog.Infof("kernel version INFO: %+v", system.HostSystemInfo)

	kubeClient := clientset.NewForConfigOrDie(config.KubeRestConf)
	klog.Infoln("kubeClient Done")
	crdClient := clientsetbeta1.NewForConfigOrDie(config.KubeRestConf)
	klog.Infoln("crdClient Done")
	topologyClient := topologyclientset.NewForConfigOrDie(config.KubeRestConf)
	klog.Infoln("topologyClient Done")
	schedulingClient := v1alpha1.NewForConfigOrDie(config.KubeRestConf)
	klog.Infoln("schedulingClient Done")

	metricCache, err := metriccache.NewMetricCache(config.MetricCacheConf)
	klog.Infoln("metricCache Done")
	if err != nil {
		return nil, err
	}
	predictServer := prediction.NewPeakPredictServer(config.PredictionConf)
	klog.Infoln("predictServer Done")
	predictorFactory := prediction.NewPredictorFactory(predictServer, config.PredictionConf.ColdStartDuration, config.PredictionConf.SafetyMarginPercent)
	klog.Infoln("predictorFactory Done")

	statesInformer := statesinformerimpl.NewStatesInformer(config.StatesInformerConf, kubeClient, crdClient, topologyClient, metricCache, nodeName, schedulingClient, predictorFactory)
	klog.Infoln("statesInformer Done")

	cgroupDriver := system.GetCgroupDriver()
	system.SetupCgroupPathFormatter(cgroupDriver)

	collectorService := metricsadvisor.NewMetricAdvisor(config.CollectorConf, statesInformer, metricCache)
	klog.Infoln("collectorService Done")

	evictVersion := "v1"
	//evictVersion, err := util.FindSupportedEvictVersion(kubeClient)
	//if err != nil {
	//	return nil, err
	//}
	klog.Infoln("evictVersion Done")

	qosManager := qosmanager.NewQOSManager(config.QOSManagerConf, scheme, kubeClient, crdClient, nodeName, statesInformer, metricCache, config.CollectorConf, evictVersion)
	klog.Infoln("qosManager Done")

	runtimeHook, err := runtimehooks.NewRuntimeHook(statesInformer, config.RuntimeHookConf, scheme, kubeClient, nodeName)
	if err != nil {
		return nil, err
	}
	klog.Infoln("runtimeHook Done")

	extensionControllers := []extension.Controller{}
	for _, initFunc := range extensionControllerInitFuncs {
		extensionControllers = append(extensionControllers, initFunc(nodeName, kubeClient, statesInformer))
	}

	d := &daemon{
		metricAdvisor:  collectorService,
		statesInformer: statesInformer,
		metricCache:    metricCache,
		qosManager:     qosManager,
		runtimeHook:    runtimeHook,
		predictServer:  predictServer,
		executor:       resourceexecutor.NewResourceUpdateExecutor(),

		extensionControllers: extensionControllers,
	}

	return d, nil
}

func (d *daemon) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	klog.Infof("Starting daemon")

	// start resource executor cache
	d.executor.Run(stopCh)

	go func() {
		if err := d.metricCache.Run(stopCh); err != nil {
			klog.Fatal("Unable to run the metric cache: ", err)
		}
	}()

	// start states informer
	go func() {
		if err := d.statesInformer.Run(stopCh); err != nil {
			klog.Fatal("Unable to run the states informer: ", err)
		}
	}()
	// wait for metric advisor sync
	if !cache.WaitForCacheSync(stopCh, d.statesInformer.HasSynced) {
		klog.Fatal("time out waiting for states informer to sync")
	}

	// start metric advisor
	go func() {
		if err := d.metricAdvisor.Run(stopCh); err != nil {
			klog.Fatal("Unable to run the metric advisor: ", err)
		}
	}()
	// wait for metric advisor sync
	if !cache.WaitForCacheSync(stopCh, d.metricAdvisor.HasSynced) {
		klog.Fatal("time out waiting for metric advisor to sync")
	}

	// start predict server
	go func() {
		if err := d.predictServer.Setup(d.statesInformer, d.metricCache); err != nil {
			klog.Fatal("Unable to setup the predict server: ", err)
		}
		if err := d.predictServer.Run(stopCh); err != nil {
			klog.Fatal("Unable to run the predict server: ", err)
		}
	}()

	// start qos manager
	go func() {
		if err := d.qosManager.Run(stopCh); err != nil {
			klog.Fatal("Unable to run the qosManager: ", err)
		}
	}()

	go func() {
		if err := d.runtimeHook.Run(stopCh); err != nil {
			klog.Fatal("Unable to run the runtimeHook: ", err)
		}
	}()

	for _, c := range d.extensionControllers {
		go func(controller extension.Controller) {
			name := controller.Name()
			klog.Infof("starting extension controller %v", name)
			if err := controller.Run(stopCh); err != nil {
				klog.Fatalf("Unable to start the extension controller %v, err: %v", name, err)
			}
		}(c)
	}

	klog.Info("Start daemon successfully")
	<-stopCh
	klog.Info("Shutting down daemon")
}
