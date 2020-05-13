package redis

import (
	"context"
	"k8s.io/apimachinery/pkg/util/intstr"

	dysprozv1alpha1 "github.com/Dysproz/guestbook-operator/pkg/apis/dysproz/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
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

var log = logf.Log.WithName("controller_redis")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Redis Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRedis{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("redis-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Redis
	err = c.Watch(&source.Kind{Type: &dysprozv1alpha1.Redis{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Redis
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dysprozv1alpha1.Redis{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to Deployments owned by Redis
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dysprozv1alpha1.Redis{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to Services owned by Redis
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dysprozv1alpha1.Redis{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRedis implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRedis{}

// ReconcileRedis reconciles a Redis object
type ReconcileRedis struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Redis object and makes changes based on the state read
// and what is in the Redis.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRedis) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Redis")

	// Fetch the Redis instance
	instance := &dysprozv1alpha1.Redis{}
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

	// Set Redis instance as the owner and controller
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

	// Create Service for Redis Master
	redisMasterService := newRedisMasterService(instance)

	// Set Redis instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, redisMasterService, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Service already exists
	foundMasterService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: redisMasterService.Name, Namespace: redisMasterService.Namespace}, foundMasterService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service redisMasterService", "Pod.Namespace", redisMasterService.Namespace, "Pod.Name", redisMasterService.Name)
		err = r.client.Create(context.TODO(), redisMasterService)
		if err != nil {
			return reconcile.Result{}, err
		}

	} else if err != nil {
		return reconcile.Result{}, err
	}

	//Create Deployment with Redis Slaves

	redisSlaves := newRedisSlavesForCR(instance)

	// Set Redis instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, redisSlaves, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	foundSlaves := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: redisSlaves.Name, Namespace: redisSlaves.Namespace}, foundSlaves)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment RedisSlaves", "Pod.Namespace", redisSlaves.Namespace, "Pod.Name", redisSlaves.Name, "Slaves count", int32(instance.Spec.Size - 1))
		err = r.client.Create(context.TODO(), redisSlaves)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	redisSlavesSize := int32(instance.Spec.Size - 1)
	if foundSlaves.Spec.Replicas != &redisSlavesSize {
		err = r.client.Update(context.TODO(), redisSlaves)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Create Service for Redis Slaves
	redisSlavesService := newRedisSlavesService(instance)

	// Set Redis instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, redisSlavesService, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Service already exists
	foundSlavesService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: redisSlavesService.Name, Namespace: redisSlavesService.Namespace}, foundSlavesService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service redisSlavesService", "Pod.Namespace", redisSlavesService.Namespace, "Pod.Name", redisSlavesService.Name)
		err = r.client.Create(context.TODO(), redisSlavesService)
		if err != nil {
			return reconcile.Result{}, err
		}

	} else if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// newRedisMasterForCR returns a redis master pod with the same name/namespace as the cr
func newRedisMasterForCR(cr *dysprozv1alpha1.Redis) *corev1.Pod {
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
					Image: cr.Spec.MasterImage,
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

func newRedisSlavesForCR(cr *dysprozv1alpha1.Redis) *appsv1.Deployment {
	slavesReplicas := int32(cr.Spec.Size - 1)
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
							Image: cr.Spec.SlaveImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 6379,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "GET_HOSTS_FROM",
									Value: "env",
								},
								{
									Name:  "REDIS_MASTER_SERVICE_HOST",
									Value: cr.Name + "-redis-master-service",
								},
							},
						},
					},
				},
			},
		},
	}
}

func newRedisMasterService(cr *dysprozv1alpha1.Redis) *corev1.Service {
	labels := map[string]string{
		"app":  cr.Name,
		"role": "master",
		"tier": "backend",
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-redis-master-service",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Port:       6379,
					TargetPort: intstr.IntOrString{IntVal: 6379},
				},
			},
			Selector: labels,
		},
	}
}

func newRedisSlavesService(cr *dysprozv1alpha1.Redis) *corev1.Service {
	labels := map[string]string{
		"app":  cr.Name,
		"role": "slave",
		"tier": "backend",
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-redis-slave-service",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Port: 6379,
				},
			},
			Selector: labels,
		},
	}
}
