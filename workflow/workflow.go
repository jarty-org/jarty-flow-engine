package workflow

import (
	"context"
	"fmt"
	workflowv1 "github.com/jartyorg.io/jarty-flow-engine/pkg/apis/workflow/v1"
	clientset "github.com/jartyorg.io/jarty-flow-engine/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ProcessWorkflow  process workflow to execute the task
func ProcessWorkflow(kubeclientset kubernetes.Interface, workflowclientset clientset.Interface, wf *workflowv1.Workflow) {
	if wf.Status.Status != workflowv1.WorkflowStatusPending {
		return
	}

	entry := findEntryStep(wf)
	if entry == nil {
		logrus.Errorf("workflow %s has no entry step", wf.Name)
		return
	}

	err := createPod(kubeclientset, wf.ObjectMeta.Namespace, wf.Name, *entry)

	if err != nil {
		logrus.Errorf("create pod error: %s", err.Error())
		return
	}

	wf.Status.Status = workflowv1.WorkflowStatusRunning
	_, err = workflowclientset.JartyorgV1().Workflows(wf.ObjectMeta.Namespace).Update(context.Background(), wf, metav1.UpdateOptions{})
	if err != nil {
		logrus.Errorf("update workflow status error: %s", err.Error())
		return
	}

}

func findEntryStep(wf *workflowv1.Workflow) *workflowv1.WorkflowStep {
	for _, step := range wf.Spec.Steps {
		if step.Name == wf.Spec.Entry {
			return &step
		}
	}
	return nil
}

func createPod(clientset kubernetes.Interface, namespace string, workflowName string, step workflowv1.WorkflowStep) error {
	podName := fmt.Sprintf("%s-%s", workflowName, step.Name)
	step.Contaienr.Name = podName

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				step.Contaienr,
			},
		},
	}

	// 使用客户端创建 Pod
	_, err := clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	return err
}
