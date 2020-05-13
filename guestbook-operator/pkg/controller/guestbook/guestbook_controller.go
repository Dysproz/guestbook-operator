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

	// Watch for changes to primary resource Redis
	err = c.Watch(&source.Kind{Type: &dysprozv1alpha1.Redis{}}, &handler.EnqueueRequestForObject{})
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

	// Fetch the Guestbook instance
	instance := &dysprozv1alpha1.Guestbook{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	redis := newRedis(instance)

	// Set Redis instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, redis, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if Redis already exists
	foundRedis := &dysprozv1alpha1.Redis{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: redis.Name, Namespace: redis.Namespace}, foundRedis)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Redis", "Pod.Namespace", redis.Namespace, "Pod.Name", redis.Name)
		err = r.client.Create(context.TODO(), redis)
		if err != nil {
			return reconcile.Result{}, err
		}

	} else if err != nil {
		return reconcile.Result{}, err
	}

	if foundRedis.Spec.Size != instance.Spec.RedisSize {
		err = r.client.Update(context.TODO(), redis)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Create Deployment with Frontend pods

	frontendPods := newFrontend(instance)

	// Set Guestbook instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, frontendPods, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	foundFrontend := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: frontendPods.Name, Namespace: frontendPods.Namespace}, foundFrontend)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment frontendPods", "Pod.Namespace", frontendPods.Namespace, "Pod.Name", frontendPods.Name)
		err = r.client.Create(context.TODO(), frontendPods)
		if err != nil {
			return reconcile.Result{}, err
		}

	} else if err != nil {
		return reconcile.Result{}, err
	}
	frontendSize := int32(instance.Spec.GuestbookSize)
	if foundFrontend.Spec.Replicas != &frontendSize {
		err = r.client.Update(context.TODO(), frontendPods)
	}

	// Create Service for Frontend
	frontendService := newFrontendService(instance)

	// Set Guestbook instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, frontendService, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Service already exists
	foundFrontendService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: frontendService.Name, Namespace: frontendService.Namespace}, foundFrontendService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service frontendService", "Pod.Namespace", frontendService.Namespace, "Pod.Name", frontendService.Name)
		err = r.client.Create(context.TODO(), frontendService)
		if err != nil {
			return reconcile.Result{}, err
		}

	} else if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func newRedis(cr *dysprozv1alpha1.Guestbook) *dysprozv1alpha1.Redis {
	labels := map[string]string{
		"app":  cr.Name + "-redis",
	}
	return &dysprozv1alpha1.Redis{
		ObjectMeta: metav1.ObjectMeta{
			Name: cr.Name + "-redis",
			Namespace: cr.Namespace,
			Labels: labels,
		},
		Spec: dysprozv1alpha1.RedisSpec{
			Size: cr.Spec.RedisSize,
			MasterImage: cr.Spec.RedisMasterImage,
			SlaveImage: cr.Spec.RedisSlaveImage,
		},
	}
}

func newFrontend(cr *dysprozv1alpha1.Guestbook) *appsv1.Deployment {
	frontendSize := int32(cr.Spec.GuestbookSize)
	labels := map[string]string{
		"app":  cr.Name,
		"tier": "frontend",
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-frontend",
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": cr.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: &frontendSize,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
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
					},
				},
			},
		},
	}
}

func newFrontendService(cr *dysprozv1alpha1.Guestbook) *corev1.Service {
	labels := map[string]string{
		"app":  cr.Name,
		"tier": "frontend",
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-frontend-service",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
			Selector: labels,
		},
	}
}
