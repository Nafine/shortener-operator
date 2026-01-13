/*
Copyright 2026.

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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	oputils "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	shortenerv1alpha1 "github.com/nafine/shortener-operator/api/v1alpha1"
)

const (
	available = "Available"
)

// UrlShortenerReconciler reconciles a UrlShortener object
type UrlShortenerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=shortener.nafine.dev,resources=urlshorteners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=shortener.nafine.dev,resources=urlshorteners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=shortener.nafine.dev,resources=urlshorteners/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *UrlShortenerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	shortener := &shortenerv1alpha1.UrlShortener{}
	if err := r.Get(ctx, req.NamespacedName, shortener); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if len(shortener.Status.Conditions) == 0 {
		meta.SetStatusCondition(&shortener.Status.Conditions, metav1.Condition{
			Type:    available,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		})
		if err := r.Status().Update(ctx, shortener); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, shortener); err != nil {
			return ctrl.Result{}, err
		}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shortener.Name,
			Namespace: shortener.Namespace,
		},
	}

	op, err := oputils.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		var desiredReplicas int32 = 0
		if shortener.Spec.Replicas != nil {
			desiredReplicas = *shortener.Spec.Replicas
		}

		if err := r.mutateDeployment(shortener, deployment, desiredReplicas); err != nil {
			return err
		}

		return ctrl.SetControllerReference(shortener, deployment, r.Scheme)
	})

	if err != nil {
		log.Error(err, "Failed to reconcile Deployment")
		meta.SetStatusCondition(&shortener.Status.Conditions, metav1.Condition{
			Type:    available,
			Status:  metav1.ConditionFalse,
			Reason:  "Reconciling",
			Message: fmt.Sprintf("Failed to reconcile deployment: %v", err),
		})
		if err := r.Status().Update(ctx, shortener); err != nil {
			log.Error(err, "Failed to update UrlShortener status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if op != oputils.OperationResultNone {
		log.Info("Deployment reconciled", "Operation", op)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shortener.Name,
			Namespace: shortener.Namespace,
		},
	}

	op, err = oputils.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Spec.Selector = map[string]string{"app.kubernetes.io/name": "project"}
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       *shortener.Spec.Port,
				TargetPort: intstr.FromInt32(*shortener.Spec.Port),
			},
		}

		return ctrl.SetControllerReference(shortener, service, r.Scheme)
	})

	if err != nil {
		log.Error(err, "Failed to reconcile Service")
		meta.SetStatusCondition(&shortener.Status.Conditions, metav1.Condition{
			Type:    available,
			Status:  metav1.ConditionFalse,
			Reason:  "Reconciling",
			Message: fmt.Sprintf("Failed to reconcile service: %v", err),
		})
		if err := r.Status().Update(ctx, shortener); err != nil {
			log.Error(err, "Failed to update UrlShortener status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if op != oputils.OperationResultNone {
		log.Info("Service reconciled", "Operation", op)
	}

	meta.SetStatusCondition(&shortener.Status.Conditions, metav1.Condition{
		Type:    available,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciling",
		Message: fmt.Sprintf("Сustom resource (%s) is ready", shortener.Name),
	})

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UrlShortenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&shortenerv1alpha1.UrlShortener{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Named("urlshortener").
		Complete(r)
}

func (r *UrlShortenerReconciler) mutateDeployment(shortener *shortenerv1alpha1.UrlShortener, dep *appsv1.Deployment, replicas int32) error {
	image := "naf1ne/url-shortener:latest"

	dep.Spec.Replicas = ptr.To(replicas)
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{"app.kubernetes.io/name": "project"},
	}
	dep.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"app.kubernetes.io/name": "project"},
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
			Containers: []corev1.Container{{
				Image:           image,
				Name:            "shortener",
				ImagePullPolicy: corev1.PullIfNotPresent,
				SecurityContext: &corev1.SecurityContext{
					RunAsNonRoot:             ptr.To(true),
					RunAsUser:                ptr.To(int64(65532)),
					AllowPrivilegeEscalation: ptr.To(false),
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
					},
				},
				Ports: []corev1.ContainerPort{{
					ContainerPort: *shortener.Spec.Port,
					Name:          "shortener",
				}},
				Env: []corev1.EnvVar{
					{Name: "APP_ENV", Value: *shortener.Spec.AppEnv},
					{Name: "STORAGE_DSN", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: shortener.Spec.StorageDsnSecretRef}},
					{Name: "HTTP_HOST", Value: "0.0.0.0"},
					{Name: "HTTP_PORT", Value: fmt.Sprintf("%d", *shortener.Spec.Port)},
					{Name: "HTTP_TIMEOUT", Value: shortener.Spec.Timeout.Duration.String()},
					{Name: "HTTP_IDLE_TIMEOUT", Value: shortener.Spec.IdleTimeout.Duration.String()},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: shortener.Name, MountPath: "/etc/shortener"},
				},
			}},
			Volumes: []corev1.Volume{{
				Name: shortener.Name,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: shortener.Spec.ApiKeysSecretRef.Name,
					},
				},
			}},
		},
	}
	return nil
}
