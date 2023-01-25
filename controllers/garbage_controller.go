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
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	examplev1 "bebc.com/garbage-collection/api/v1"
)

const (
	defaultReplicas = int32(1)
	defaultImages   = "nginx"
	eventCreate     = "create"
	eventUpdate     = "update"
)

type GarbagePredicate struct {
	predicate.Funcs
}

// GarbageReconciler reconciles a Garbage object
type GarbageReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//func (p *GarbagePredicate) Update(e event.UpdateEvent) bool {
//	obj := e.ObjectNew.GetObjectKind()
//	fmt.Println(obj.GroupVersionKind().Kind)
//	//oldGarbage := e.ObjectOld.(*examplev1.Garbage)
//	newGarbage, ok := e.ObjectNew.(*examplev1.Garbage)
//	if ok && newGarbage.DeletionTimestamp != nil {
//		return false
//	}
//
//	//if reflect.DeepEqual(oldGarbage.Spec, newGarbage.Spec) {
//	//	return false
//	//}
//
//	return true
//}

//+kubebuilder:rbac:groups=example.bebc.com,resources=garbages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=example.bebc.com,resources=garbages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=example.bebc.com,resources=garbages/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Garbage object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *GarbageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	garbage := &examplev1.Garbage{}

	if err := r.Client.Get(context.TODO(), req.NamespacedName, garbage); err != nil {
		// The resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Info("get garbage", "error", err)
		return ctrl.Result{}, fmt.Errorf("get garbage %v err %w", req.Name, err)
	}

	if garbage.DeletionTimestamp != nil {
		if r.hasFinalizer(garbage) {
			//过10s删除
			err := r.deleteExternalResources(garbage)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("delete external resourceerr %w", err)
			}
			r.removeFinalizer(garbage)
			//meta := metav1.ObjectMeta{
			//	Finalizers: garbage.Finalizers,
			//}
			//b, err := json.Marshal(meta)
			//if err != nil {
			//	return ctrl.Result{}, fmt.Errorf("marshal err %v", err)
			//}
			//patch = []byte(`{"metadata":{"finalizers":{"version": "v2"}}}` + patch + "}`")

			//logger.Info("patch", "value", string(b))
			err = r.Update(ctx, garbage)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("remove finalizer and update garbage err %w", err)
			}

		}

		return ctrl.Result{}, nil
	}

	if garbage.Spec.Nginx == nil {
		return ctrl.Result{}, fmt.Errorf("garbage.spec.nginx is nil")
	}

	obj := garbage.DeepCopy()

	//添加finalizer
	if obj.Spec.SetFinalizer.Set {
		r.addFinalizer(obj)
	} else {
		r.removeFinalizer(obj)
	}

	err := r.Update(ctx, obj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("update garbage finalizer err %w", err)
	}

	eventType, err := r.createOrUpdateNginx(ctx, obj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("create or update garbage err %w", err)
	}
	r.Recorder.Eventf(garbage, corev1.EventTypeNormal, eventType, "Successfully %v", eventType)

	err = r.updateStatus(ctx, obj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("update garbage status err %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *GarbageReconciler) createOrUpdateNginx(ctx context.Context, obj *examplev1.Garbage) (string, error) {
	oldDeploy := &appsv1.Deployment{}
	newDeploy := r.getDeployNginx(obj)
	if err := r.Get(ctx, types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name + "-example"}, oldDeploy); err != nil {
		if apierrors.IsNotFound(err) {
			if obj.Spec.SetOwn {
				r.addOwnReference(obj, newDeploy)
			}
			if obj.Spec.SetFinalizer.Set {
				r.addFinalizer(obj)
			}
			err = r.Create(ctx, newDeploy)
			if err != nil {
				return eventCreate, err
			}

			return eventCreate, nil
		}
		return eventCreate, err
	}

	if !reflect.DeepEqual(oldDeploy.Spec, newDeploy.Spec) {
		oldDeploy.Spec = newDeploy.Spec
		if obj.Spec.SetOwn {
			r.addOwnReference(obj, oldDeploy)
		} else {
			r.removeOwnReference(obj, oldDeploy)
		}

		if obj.Spec.SetFinalizer.Set {
			r.addFinalizer(obj)
		} else {
			r.removeFinalizer(obj)
		}

		err := r.Update(ctx, oldDeploy)
		if err != nil {
			return eventUpdate, err
		}
	}

	return eventUpdate, nil
}

func (r *GarbageReconciler) updateStatus(ctx context.Context, obj *examplev1.Garbage) error {
	var deployment appsv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name + "-example"}, &deployment);
		err != nil {
		return client.IgnoreNotFound(err)
	}

	availableReplicas := deployment.Status.AvailableReplicas
	if availableReplicas == obj.Status.AvailableReplicas {
		return nil
	}
	obj.Status.AvailableReplicas = availableReplicas

	var (
		availableCondition = examplev1.Condition{
			Type:   examplev1.Available,
			Status: examplev1.ConditionTrue,
			LastTransitionTime: metav1.Time{
				Time: time.Now().UTC(),
			},
			ObservedGeneration: obj.Generation,
		}
	)

	if availableReplicas == 0 {
		availableCondition.Reason = "NoPodReady"
		availableCondition.Status = examplev1.ConditionFalse
	} else if availableReplicas != *deployment.Spec.Replicas {
		availableCondition.Reason = "SomePodsNotReady"
		availableCondition.Status = examplev1.ConditionDegraded
	}
	obj.Status.Conditions = append(obj.Status.Conditions, availableCondition)
	if err := r.Status().Update(ctx, obj); err != nil {
		return err
	}

	return nil
}

func (r *GarbageReconciler) getDeployNginx(obj *examplev1.Garbage) *appsv1.Deployment {
	var image *string
	var replica *int32

	if obj.Spec.Nginx.Image == nil {
		*image = defaultImages
	}

	if obj.Spec.Nginx.Replica == nil {
		*replica = defaultReplicas
	}

	image = obj.Spec.Nginx.Image
	replica = obj.Spec.Nginx.Replica

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.Name + "-example",
			Namespace: obj.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": obj.Name + "-example",
				},
			},
			Replicas: replica,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": obj.Name + "-example",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: *image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "HOST_IP_ADDRESS",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *GarbageReconciler) addOwnReference(obj *examplev1.Garbage, deploy *appsv1.Deployment) {
	owner := metav1.NewControllerRef(obj, examplev1.GroupVersion.WithKind(examplev1.ResourceKindGarbage))
	if deploy.OwnerReferences != nil {
		for i := range deploy.OwnerReferences {
			if deploy.OwnerReferences[i].Name == obj.Name {
				return
			}
		}
		//deploy.OwnerReferences = append(deploy.OwnerReferences, *owner)
		//return
	}

	deploy.OwnerReferences = append(deploy.OwnerReferences, *owner)
}

func (r *GarbageReconciler) addFinalizer(obj *examplev1.Garbage) {
	if len(obj.Finalizers) != 0 {
		for _, name := range obj.Finalizers {
			if name == *obj.Spec.SetFinalizer.Name {
				return
			}
		}
	}

	finalizerName := obj.Spec.SetFinalizer.Name
	obj.Finalizers = append(obj.Finalizers, *finalizerName)
}

func (r *GarbageReconciler) removeFinalizer(obj *examplev1.Garbage) {
	if len(obj.Finalizers) != 0 {
		for i, name := range obj.Finalizers {
			if name == *obj.Spec.SetFinalizer.Name {
				obj.Finalizers = append(obj.Finalizers[:i], obj.Finalizers[i+1:]...)
				return
			}
		}
	}
}

func (r *GarbageReconciler) removeOwnReference(obj *examplev1.Garbage, deploy *appsv1.Deployment) {
	//owner := metav1.NewControllerRef(obj, examplev1.GroupVersion.WithKind(examplev1.ResourceKindGarbage))
	if deploy.OwnerReferences != nil {
		for i := range deploy.OwnerReferences {
			deploy.OwnerReferences[i].Name = obj.Name
			//ownerReverence.Name = obj.Name
			deploy.OwnerReferences = append(deploy.OwnerReferences[:i], deploy.OwnerReferences[i+1:]...)
		}
	}
	//deploy.OwnerReferences = []metav1.OwnerReference{}
}

func (r *GarbageReconciler) hasFinalizer(obj *examplev1.Garbage) bool {
	if len(obj.Finalizers) == 0 {
		return false
	}
	for _, name := range obj.Finalizers {
		if name == *obj.Spec.SetFinalizer.Name {
			return true
		}
	}
	return false
}

func (r *GarbageReconciler) deleteExternalResources(obj any) error {
	time.Sleep(10 * time.Second)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GarbageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1.Garbage{}).Owns(&appsv1.Deployment{}).WithEventFilter(&GarbagePredicate{}).
		Complete(r)
}
