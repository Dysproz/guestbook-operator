package guestbook

import (
	"context"

	dysprozv1alpha1 "github.com/Dysproz/guestbook-operator/pkg/apis/dysproz/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_guestbook")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Guestbook Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileGuestbook{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("guestbook-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Guestbook
	err = c.Watch(&source.Kind{Type: &dysprozv1alpha1.Guestbook{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Redis
	err = c.Watch(&source.Kind{Type: &dysprozv1alpha1.Redis{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dysprozv1alpha1.Guestbook{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to Pods owned by Guestbook
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dysprozv1alpha1.Guestbook{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to Deployments owned by Guestbook
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dysprozv1alpha1.Guestbook{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to Services owned by Guestbook
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dysprozv1alpha1.Guestbook{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileGuestbook implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileGuestbook{}

// ReconcileGuestbook reconciles a Guestbook object
type ReconcileGuestbook struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Guestbook object and makes changes based on the state read
// and what is in the Guestbook.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGuestbook) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Guestbook")

	ctx := context.TODO()

	// Fetch the Guestbook instance
	instance := &dysprozv1alpha1.Guestbook{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Create Redis

	var redis dysprozv1alpha1.Redis
	redis.Name = instance.Name + "-redis"
	redis.Namespace = instance.Namespace

	_, err = ctrl.CreateOrUpdate(ctx, r.client, &redis, func() error {
		modifyRedis(instance, &redis)
		return controllerutil.SetControllerReference(instance, &redis, r.scheme)
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	updatedRedis := &dysprozv1alpha1.Redis{}
	_ = r.client.Get(ctx, types.NamespacedName{Name: redis.Name, Namespace: redis.Namespace}, updatedRedis)
	if !updatedRedis.Status.Ready {
		return reconcile.Result{Requeue: true}, nil
	}

	// Create Deployment with Frontend pods
	var frontend appsv1.Deployment
	frontend.Name = instance.Name + "-frontend"
	frontend.Namespace = instance.Namespace
	_, err = ctrl.CreateOrUpdate(ctx, r.client, &frontend, func() error {
		modifyFrontend(instance, &frontend)
		return controllerutil.SetControllerReference(instance, &frontend, r.scheme)
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create Service for Frontend
	var frontSvc corev1.Service
	frontSvc.Name = instance.Name + "-frontend-service"
	frontSvc.Namespace = instance.Namespace
	_, err = ctrl.CreateOrUpdate(ctx, r.client, &frontSvc, func() error {
		modifyFrontendService(instance, &frontSvc)
		return controllerutil.SetControllerReference(instance, &frontSvc, r.scheme)
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	if redis.Status.Ready && frontend.Status.ReadyReplicas == int32(instance.Spec.GuestbookSize) {
		reqLogger.Info("Guestbook is ready")
		instance.Status.Ready = true
	} else {
		reqLogger.Info("Wainitng for Guestbook to become ready")
		instance.Status.Ready = false
	}

	if err := r.client.Status().Update(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func modifyRedis(cr *dysprozv1alpha1.Guestbook, redis *dysprozv1alpha1.Redis) {
	labels := map[string]string{
		"app": cr.Name + "-redis",
	}
	if redis.Labels == nil {
		redis.Labels = labels
	}
	redis.Spec.Size = cr.Spec.RedisSize
	redis.Spec.MasterImage = cr.Spec.RedisMasterImage
	redis.Spec.SlaveImage = cr.Spec.RedisSlaveImage
}

func modifyFrontend(cr *dysprozv1alpha1.Guestbook, front *appsv1.Deployment) {
	frontendSize := int32(cr.Spec.GuestbookSize)
	labels := map[string]string{
		"app":  cr.Name,
		"tier": "frontend",
	}
	if front.ObjectMeta.Labels == nil {
		front.ObjectMeta.Labels = labels
	}
	if front.Spec.Template.ObjectMeta.Labels == nil {
		front.Spec.Template.ObjectMeta.Labels = labels
	}
	if front.ObjectMeta.Labels == nil {
		front.ObjectMeta.Labels = map[string]string{
			"app": cr.Name,
		}
	}
	front.Spec.Replicas = &frontendSize
	front.Spec.Template.ObjectMeta.Labels = labels
	front.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}
	front.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:  cr.Name + "-php-frontend",
			Image: cr.Spec.GuestbookImage,
			Env: []corev1.EnvVar{
				{
					Name:  "GET_HOSTS_FROM",
					Value: "env",
				},
				{
					Name:  "REDIS_SLAVE_SERVICE_HOST",
					Value: cr.Name + "-redis-slave-service",
				},
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 6379,
				},
			},
		},
	}
}

func modifyFrontendService(cr *dysprozv1alpha1.Guestbook, frontService *corev1.Service) {
	labels := map[string]string{
		"app":  cr.Name,
		"tier": "frontend",
	}
	if frontService.ObjectMeta.Labels == nil {
		frontService.ObjectMeta.Labels = labels
	}
	frontService.Spec.Type = corev1.ServiceTypeNodePort
	frontService.Spec.Ports = []corev1.ServicePort{
		{
			Port: 80,
		},
	}
	frontService.Spec.Selector = labels
}
