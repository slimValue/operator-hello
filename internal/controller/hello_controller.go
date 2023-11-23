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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	demov1 "mega.crd/demo/api/v1"
)

// HelloReconciler reconciles a Hello object
type HelloReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=demo.mega.crd,resources=hellos,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.mega.crd,resources=hellos/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.mega.crd,resources=hellos/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Hello object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *HelloReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	// add slim
	/**
	接下来实现控制器逻辑。没什么复杂的，通过触发 reconciliation 请求获取 Foo 资源，从而得到 Foo 的朋友的名称。然后，列出所有和 Foo 的朋友同名的 Pod。如果找到一个或多个，将 Foo 的 happy 状态更新为 true，否则设置为 false。
	注意，控制器也会对 Pod 事件做出反应。实际上，如果创建了一个新的 Pod，我们希望 Foo 资源能够相应更新其状态。这个方法将在每次发生 Pod 事件时被触发（创建、更新或删除）。然后，只有当 Pod 名称是集群中部署的某个 Foo 自定义资源的“朋友”时，才触发 Foo 控制器的 reconciliation 循环。
	*/
	log := log.FromContext(ctx)
	log.Info("reconciling demo custom resource")
	var foo demov1.Hello
	if err := r.Get(ctx, req.NamespacedName, &foo); err != nil {
		log.Error(err, "unable to fetch Foo")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get pods with the same name as Foo's friend
	var podList corev1.PodList
	var friendFound bool
	if err := r.List(ctx, &podList); err != nil {
		log.Error(err, "unable to list pods")
	} else {
		for _, item := range podList.Items {
			if item.GetName() == foo.Spec.Name {
				log.Info("pod linked to a foo custom resource found", "name", item.GetName())
				friendFound = true
			}
		}
	}

	// Update Foo' happy status
	foo.Status.Happy = friendFound
	if err := r.Status().Update(ctx, &foo); err != nil {
		log.Error(err, "unable to update foo's happy status", "status", friendFound)
		return ctrl.Result{}, err
	}
	log.Info("demo's happy status updated", "status", friendFound)

	log.Info("demo custom resource reconciled")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelloReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.Hello{}).
		Watches(
			//&source.Kind{Type: &corev1.Pod{}},
			//source.Kind(mgr.GetCache(),  &corev1.Pod{}),
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.mapPodsReqToFooReq),
		).
		Complete(r)
}

func (r *HelloReconciler) mapPodsReqToFooReq(ctx context.Context, obj client.Object) []reconcile.Request {
	log := log.FromContext(ctx)

	// List all the Foo custom resource
	req := []reconcile.Request{}
	var list demov1.HelloList
	if err := r.Client.List(context.TODO(), &list); err != nil {
		log.Error(err, "unable to list foo custom resources")
	} else {
		// Only keep Foo custom resources related to the Pod that triggered the reconciliation request
		for _, item := range list.Items {
			if item.Spec.Name == obj.GetName() {
				req = append(req, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: item.Name, Namespace: item.Namespace},
				})
				log.Info("pod linked to a foo custom resource issued an event", "name", obj.GetName())
			}
		}
	}
	return req
}
