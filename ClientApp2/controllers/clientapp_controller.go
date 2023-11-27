/*
Copyright 2023.

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

package controllers

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	customv1alpha1 "github.com/clinia/clientapp-operator/api/v1alpha1"
)

// ClientappReconciler reconciles a Clientapp object
type ClientappReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=custom.clinia.com,resources=clientapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=custom.clinia.com,resources=clientapps/status,verbs=get;update;patch;watch
//+kubebuilder:rbac:groups=custom.clinia.com,resources=clientapps/finalizers,verbs=update;watch

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Clientapp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ClientappReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// TODO(user): your logic here
	clientApp := &customv1alpha1.Clientapp{}
	err := r.Get(ctx, req.NamespacedName, clientApp)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{Requeue: true}, nil
		}
	}
	// Check if the deployment already exists, if not create a new deployment.
	logger.Info("Operator checking if deployment already exists.")
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: clientApp.Name, Namespace: clientApp.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			// Define and create a new deployment.
			dep := r.CreateClientApp(ctx, clientApp)
			if err = r.Create(ctx, dep); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Ensure the deployment size is the same as the spec.
	size := clientApp.Spec.Replicas
	if *found.Spec.Replicas != size {
		logger.Info(fmt.Sprintf("The desired: %v replicas not equal to actual: %v", size, *found.Spec.Replicas))
		found.Spec.Replicas = &size
		if err = r.Update(ctx, found); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Ensure the resources defined and actual are same
	requestCPU, existsCPU := clientApp.Spec.Resources[corev1.ResourceCPU]
	requestMemory, existsMemory := clientApp.Spec.Resources[corev1.ResourceMemory]
	containers := &found.Spec.Template.Spec.Containers[0] // assuming for only container, If there are many need to change accordingly
	logger.Info(fmt.Sprintf("Checking the desired state CPU:%v Memory:%v and actual state for resources", requestCPU, requestMemory))

	if containers.Resources.Requests == nil {
		containers.Resources.Requests = make(corev1.ResourceList)
	}
	if existsCPU && existsMemory {
		if containers.Resources.Requests[corev1.ResourceCPU] != requestCPU ||
			containers.Resources.Requests[corev1.ResourceMemory] != requestMemory {
			logger.Info(fmt.Sprintf("Resource Update : Desired state is different for Resources"))
			containers.Resources.Requests[corev1.ResourceCPU] = requestCPU
			containers.Resources.Requests[corev1.ResourceMemory] = requestMemory
			if err = r.Update(ctx, found); err != nil {
				logger.Error(err, "Failed to update the resource")
				return ctrl.Result{Requeue: true}, err
			}
			logger.Info("Resource update successful")
		}
	}

	// Ensure ENV variables are updated.
	if containers.Env == nil {
		containers.Env = make([]corev1.EnvVar, 0)
	}
	containers.Env = clientApp.Spec.Env
	if err := r.Update(ctx, found); err != nil {
		logger.Error(err, "Failed to update ClientApp env variables")
		return ctrl.Result{Requeue: true}, err
	}
	logger.Info("Updated the ClientApp env variables successfully")
	if !envVarSliceEqual(clientApp.Spec.Env, containers.Env) {
		containers.Env = clientApp.Spec.Env
		// Update the Deployment with the modified pod spec
		if err := r.Update(ctx, found); err != nil {
			logger.Error(err, "Failed to update ClientApp env variables")
			return ctrl.Result{Requeue: true}, err
		}
		logger.Info("Updated the ClientApp env variables successfully")
		// Return a requeue to allow time for the update to take effect
	}
	// if !reflect.DeepEqual(clientApp.Spec.Env, containers.Env) {
	// 	containers.Env = clientApp.Spec.Env
	// 	// Update the Deployment with the modified pod spec
	// 	if err = r.Update(ctx, found); err != nil {
	// 		logger.Error(err, "Failed to update ClientApp env variables")
	// 		return ctrl.Result{Requeue: true}, err
	// 	}
	// 	logger.Info("updated the clientApp env variables succesfully")
	// 	// Return a requeue to allow time for the update to take effect
	// 	return ctrl.Result{Requeue: true}, nil
	// }

	//Perform HealthCheck and update the Status
	healthCheckPassed := r.performHealthCheck(ctx, clientApp)
	clientApp.Status.Available = false

	if !healthCheckPassed {
		logger.Info("Health check failed. Setting Available to false.")
		clientApp.Status.Available = false
		logger.Info("updated the clientApp status succesfully")
	} else {
		// Update status with URL and set Available to true
		clientApp.Status.URL = fmt.Sprintf("http://%s:%d", clientApp.Spec.Host, clientApp.Spec.Port)
		clientApp.Status.Available = true
	}
	// update the status
	if err := r.Status().Update(ctx, clientApp); err != nil {
		logger.Error(err, "Failed to update ClientApp status")
		return ctrl.Result{Requeue: true}, nil
	}
	logger.Info("ClientApp Status updated successfully")
	return ctrl.Result{}, nil
}

// create a new deployment if not exists.
func (r *ClientappReconciler) CreateClientApp(ctx context.Context, m *customv1alpha1.Clientapp) *appsv1.Deployment {
	logger := log.FromContext(ctx)
	//	resource := m.Spec.Resources
	lbls := labelsForApp(m.Name)
	replicas := m.Spec.Replicas

	// Access the ResourceList field
	requestCPU := m.Spec.Resources.Cpu()
	requestMemory := m.Spec.Resources.Memory()
	logger.Info(fmt.Sprintf("Request CPU: %v Requested Memory: %v", requestCPU, requestMemory))

	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(requestCPU.String()),
			corev1.ResourceMemory: resource.MustParse(requestMemory.String()),
		},
	}

	envList := m.Spec.Env

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: lbls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lbls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: m.Spec.Image,
						Name:  "nginx",
						Ports: []corev1.ContainerPort{{
							ContainerPort: m.Spec.Port,
						}},
						Resources: resources,
						Env:       envList,
					}},
				},
			},
		},
	}

	return dep
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClientappReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&customv1alpha1.Clientapp{}).
		Complete(r)
}

// perform HealthCheck for the ClientApp
func (r *ClientappReconciler) performHealthCheck(ctx context.Context, m *customv1alpha1.Clientapp) bool {
	logger := log.FromContext(ctx)
	healthCheckURL := fmt.Sprintf("http://%s:%d/health", m.Spec.Host, m.Spec.Port)
	logger.Info(fmt.Sprintf("HealthCheck URL is : %s", healthCheckURL))
	client := &http.Client{
		Timeout: 5 * time.Second, // Set a timeout for the HTTP request
	}
	resp, err := client.Get(healthCheckURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func labelsForApp(name string) map[string]string {
	return map[string]string{"cr_name": name}
}

// env comparision
func envVarSliceEqual(slice1, slice2 []corev1.EnvVar) bool {
	if len(slice1) != len(slice2) {
		return false
	}

	for i := range slice1 {
		if slice1[i].Name != slice2[i].Name || slice1[i].Value != slice2[i].Value {
			return false
		}
	}

	return true
}
