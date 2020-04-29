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

	// Define a new Pod object with Redis Master
	redisMaster := newRedisMasterForCR(instance)

	// Set Guestbook instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, redisMaster, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	foundMaster := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: redisMaster.Name, Namespace: redisMaster.Namespace}, foundMaster)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod RedisMaster", "Pod.Namespace", redisMaster.Namespace, "Pod.Name", redisMaster.Name)
		err = r.client.Create(context.TODO(), redisMaster)
		if err != nil {
			return reconcile.Result{}, err
		}

	} else if err != nil {
		return reconcile.Result{}, err
	}

	//Create Deployment with Redis Slaves

	redisSlaves := newRedisSlavesForCR(instance)

	// Set Guestbook instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, redisSlaves, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	foundSlaves := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: redisSlaves.Name, Namespace: redisSlaves.Namespace}, foundSlaves)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment RedisSlaves", "Pod.Namespace", redisSlaves.Namespace, "Pod.Name", redisSlaves.Name)
		err = r.client.Create(context.TODO(), redisSlaves)
		if err != nil {
			return reconcile.Result{}, err
		}

	} else if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Skip reconcile: Pod Redis Master already exists", "Pod.Namespace", foundMaster.Namespace, "Pod.Name", foundMaster.Name)
	reqLogger.Info("Skip reconcile: Deployment Redis Slaves already exists", "Deployment.Namespace", foundSlaves.Namespace, "Deployment.Name", foundSlaves.Name)

	return reconcile.Result{}, nil
}

// newRedisMasterForCR returns a redis master pod with the same name/namespace as the cr
func newRedisMasterForCR(cr *dysprozv1alpha1.Guestbook) *corev1.Pod {
	labels := map[string]string{
		"app":  cr.Name,
		"role": "master",
		"tier": "backend",
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-redis-master",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  cr.Name + "-redis-master",
					Image: cr.Spec.RedisMasterImage,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 6379,
						},
					},
				},
			},
		},
	}
}

func newRedisSlavesForCR(cr *dysprozv1alpha1.Guestbook) *appsv1.Deployment {
	slavesReplicas := int32(cr.Spec.RedisSize - 1)
	labels := map[string]string{
		"app":  cr.Name,
		"role": "slave",
		"tier": "backend",
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-redis-slave",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &slavesReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  cr.Name + "-redis-slave",
							Image: cr.Spec.RedisSlaveImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 6379,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "GET_HOSTS_FROM",
									Value: "dns",
								},
							},
						},
					},
				},
			},
		},
	}
}
