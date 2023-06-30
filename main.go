package main

import (
	"github.com/jartyorg.io/jarty-flow-engine/config"
	worflowcontroller "github.com/jartyorg.io/jarty-flow-engine/controller"
	clientset "github.com/jartyorg.io/jarty-flow-engine/pkg/client/clientset/versioned"
	informers "github.com/jartyorg.io/jarty-flow-engine/pkg/client/informers/externalversions"
	"github.com/jartyorg.io/jarty-flow-engine/signals"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"time"
)

func main() {

	config.LoadConfig()

	stopCh := signals.SetupSignalHandler()

	// 处理入参
	cfg, err := clientcmd.BuildConfigFromFlags("", "/Users/kayshen/.kube/config")
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logrus.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	workflowclient, err := clientset.NewForConfig(cfg)
	if err != nil {
		logrus.Fatalf("Error building example clientset: %s", err.Error())
	}

	studentInformerFactory := informers.NewSharedInformerFactory(workflowclient, time.Second*30)

	//得到controller
	controller := worflowcontroller.NewController(kubeClient, workflowclient,
		studentInformerFactory.Jartyorg().V1().Workflows())

	//启动informer
	go studentInformerFactory.Start(stopCh)

	//controller开始处理消息
	if err = controller.Run(2, stopCh); err != nil {
		logrus.Fatalf("Error running controller: %s", err.Error())
	}
}
