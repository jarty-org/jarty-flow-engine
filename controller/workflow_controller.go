package controller

import (
	"fmt"
	workflowv1 "github.com/jartyorg.io/jarty-flow-engine/pkg/apis/workflow/v1"
	clientset "github.com/jartyorg.io/jarty-flow-engine/pkg/client/clientset/versioned"
	workflwscheme "github.com/jartyorg.io/jarty-flow-engine/pkg/client/clientset/versioned/scheme"
	informers "github.com/jartyorg.io/jarty-flow-engine/pkg/client/informers/externalversions/workflow/v1"
	listers "github.com/jartyorg.io/jarty-flow-engine/pkg/client/listers/workflow/v1"
	wf "github.com/jartyorg.io/jarty-flow-engine/workflow"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"time"
)

// controllerAgentName is the name of this controller
const controllerAgentName = "workflow-controller"

const (
	SuccessSynced = "Synced"

	MessageResourceSynced = "Workflow synced successfully"
)

type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	// workflowclientset is a clientset for our own API group
	workflowclientset clientset.Interface

	// workflowLister is able to list/get workflow from a shared informer's store
	workflowLister listers.WorkflowLister

	// workflowsSynced returns true if the workflow store has been synced at least once
	workflowSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be processed instead of performing it as soon as a change happens.
	workqueue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new workflow controller
func NewController(
	kubeclientset kubernetes.Interface,
	wrokflowclientset clientset.Interface,
	workflowInformer informers.WorkflowInformer) *Controller {

	runtime.Must(workflwscheme.AddToScheme(scheme.Scheme))
	logrus.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logrus.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:     kubeclientset,
		workflowclientset: wrokflowclientset,
		workflowLister:    workflowInformer.Lister(),
		workflowSynced:    workflowInformer.Informer().HasSynced,
		workqueue: workqueue.NewRateLimitingQueueWithConfig(workqueue.DefaultControllerRateLimiter(), workqueue.RateLimitingQueueConfig{
			Name: "Workflows",
		}),
		recorder: recorder,
	}

	logrus.Info("Setting up event handlers")
	// Set up an event handler for when Workflow resources change
	workflowInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueWorkflow,
		UpdateFunc: func(old, new interface{}) {
			oldWorkflow := old.(*workflowv1.Workflow)
			newWorkflow := new.(*workflowv1.Workflow)
			if oldWorkflow.ResourceVersion == newWorkflow.ResourceVersion {
				return
			}
			controller.enqueueWorkflow(new)
		},
		DeleteFunc: controller.enqueueWorkflowForDelete,
	})

	return controller
}

// Run start the workflow controller
func (c *Controller) Run(thread int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	logrus.Info("Starting workflow controller")
	if ok := cache.WaitForCacheSync(stopCh, c.workflowSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logrus.Info("Starting workers")
	for i := 0; i < thread; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	logrus.Info("Started workers")
	<-stopCh
	logrus.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {

	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {

			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// 在syncHandler中处理业务
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}

		c.workqueue.Forget(obj)
		logrus.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// 处理
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// 从缓存中取对象
	workflow, err := c.workflowLister.Workflows(namespace).Get(name)
	if err != nil {
		// if Workflow object has been deleted, just log the event and return
		if errors.IsNotFound(err) {
			logrus.Infof("Workflow对象被删除，请在这里执行实际的删除业务: %s/%s ...", namespace, name)
			return nil
		}
		runtime.HandleError(fmt.Errorf("failed to list workflow by: %s/%s", namespace, name))
		return err
	}

	if workflow != nil {
		wf.ProcessWorkflow(c.kubeclientset, c.workflowclientset, workflow)
	}

	logrus.Infof("这里是workflow对象的期望状态: %#v ...", workflow)
	logrus.Infof("实际状态是从业务层面得到的，此处应该去的实际状态，与期望状态做对比，并根据差异做出响应(新增或者删除)")

	c.recorder.Event(workflow, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// put key into workqueue
func (c *Controller) enqueueWorkflow(obj interface{}) {
	var key string
	var err error
	// 将对象放入缓存
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}

	// 将key放入队列
	c.workqueue.AddRateLimited(key)
}

// put key into workqueue for delete
func (c *Controller) enqueueWorkflowForDelete(obj interface{}) {
	var key string
	var err error

	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}

	// add key to queue
	c.workqueue.AddRateLimited(key)
}
