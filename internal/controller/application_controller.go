/*
Copyright 2025.

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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/pachirode/operator-demo/api/v1"
)

const (
	APP_FINALIZER = "pachirode.com/app"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.crd.pachirode.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.crd.pachirode.com,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.crd.pachirode.com,resources=applications/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	log := logger.WithValues("application", req.NamespacedName)

	log.Info("start reconcile")

	var app v1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		log.Error(err, "unable to fetch Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 当前没有接收到删除请求，查看 finalizer 是否存在
	if app.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&app, APP_FINALIZER) {
			controllerutil.AddFinalizer(&app, APP_FINALIZER)
			if err := r.Update(ctx, &app); err != nil {
				log.Error(err, "unable to add finalizer to application")
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(&app, APP_FINALIZER) {
			// 检测到存在 finalizer 需要先做清理
			log.Info("start cleanup")

			// 移除 finalizer
			log.Info("start remove finalizer")
			controllerutil.RemoveFinalizer(&app, APP_FINALIZER)
			if err := r.Update(ctx, &app); err != nil {
				return ctrl.Result{}, err
			}
		}

		// 资源开始删除，退出检测
		return ctrl.Result{}, nil
	}

	// 调谐逻辑
	if err := r.syncApp(ctx, app); err != nil {
		log.Error(err, "unable to sync application")
		return ctrl.Result{}, err
	}

	// 检查状态
	var deploy appsv1.Deployment
	objKey := client.ObjectKey{Namespace: app.Namespace, Name: app.Name}
	if err := r.Get(ctx, objKey, &deploy); err != nil {
		log.Error(err, "unable to fetch deployment", "deployment", objKey.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	copyApp := app.DeepCopy()
	copyApp.Status.Ready = deploy.Status.ReadyReplicas > 0
	if err := r.Status().Update(ctx, copyApp); err != nil {
		log.Error(err, "unable to update application status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Application{}).
		Owns(&appsv1.Deployment{}).
		Named("application").
		Complete(r)
}
