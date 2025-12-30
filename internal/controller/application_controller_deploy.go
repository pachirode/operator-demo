package controller

import (
	"context"

	pkgerror "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/pachirode/operator-demo/api/v1"
)

func (r *ApplicationReconciler) syncApp(ctx context.Context, app v1.Application) error {
	if app.Spec.Enabled {
		return r.syncAppEnabled(ctx, app)
	}
	return r.syncAppDisable(ctx, app)
}

func (r *ApplicationReconciler) syncAppDisable(ctx context.Context, app v1.Application) error {
	logger := logf.FromContext(ctx)
	log := logger.WithValues("app", app.Namespace)

	var deploy appsv1.Deployment
	objKey := client.ObjectKey{Namespace: app.Namespace, Name: app.Name}
	err := r.Get(ctx, objKey, &deploy)
	if err != nil {
		// 资源已经被删除
		if errors.IsNotFound(err) {
			return nil
		}
		return pkgerror.WithMessagef(err, "unable to fetch deployment %s", objKey.String())
	}

	log.Info("reconcile application delete deployment", "deployment", objKey.Name)

	if err = r.Delete(ctx, &deploy); err != nil {
		return pkgerror.WithMessage(err, "unable to delete deployment")
	}
	return nil
}

func (r *ApplicationReconciler) syncAppEnabled(ctx context.Context, app v1.Application) error {
	logger := logf.FromContext(ctx)
	log := logger.WithValues("app", app.Namespace)

	var deploy appsv1.Deployment
	objKey := client.ObjectKey{Namespace: app.Namespace, Name: app.Name}
	err := r.Get(ctx, objKey, &deploy)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("reconcile application create deployment", "deployment", objKey.Name)
			deploy = r.generateDeployment(app)
			if err = r.Create(ctx, &deploy); err != nil {
				return pkgerror.WithMessage(err, "unable to create deployment")
			}
		}
		return pkgerror.WithMessagef(err, "unable to fetch deployment [%s]", objKey.String())
	}
	if !equal(app, deploy) {
		log.Info("reconcile application update deployment", "deployment", objKey.Name)
		deploy.Spec.Template.Spec.Containers[0].Image = app.Spec.Image
		if err = r.Update(ctx, &deploy); err != nil {
			return pkgerror.WithMessage(err, "unable to update deployment")
		}
	}
	return nil
}

func (r *ApplicationReconciler) generateDeployment(app v1.Application) appsv1.Deployment {
	deploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels: map[string]string{
				"app": app.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": app.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            app.Name,
							Image:           app.Spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		},
	}
	// 设置 ownerRef 确保应用删除时，deployment 也会被删除
	_ = controllerutil.SetControllerReference(&app, &deploy, r.Scheme)
	return deploy
}

func equal(app v1.Application, deploy appsv1.Deployment) bool {
	return deploy.Spec.Template.Spec.Containers[0].Image == app.Spec.Image
}
