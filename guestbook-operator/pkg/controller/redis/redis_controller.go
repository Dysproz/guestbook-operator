package redis

import (
	"context"
	"k8s.io/apimachinery/pkg/util/intstr"

	dysprozv1alpha1 "github.com/Dysproz/guestbook-operator/pkg/apis/dysproz/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

	ctx := context.TODO()

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
	var masterPod corev1.Pod
	masterPod.Name = instance.Name + "-redis-master"
	masterPod.Namespace = instance.Namespace
	_, err = ctrl.CreateOrUpdate(ctx, r.client, &masterPod, func() error {
		modifyRedisMasterForCR(instance, &masterPod)
		return controllerutil.SetControllerReference(instance, &masterPod, r.scheme)
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create Service for Redis Master
	var masterSvc corev1.Service
	masterSvc.Name = instance.Name + "-redis-master-service"
	masterSvc.Namespace = instance.Namespace
	_, err = ctrl.CreateOrUpdate(ctx, r.client, &masterSvc, func() error {
		modifyRedisMasterService(instance, &masterSvc)
		return controllerutil.SetControllerReference(instance, &masterSvc, r.scheme)
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	if masterPod.Status.Phase != "Running" {
		reqLogger.Info("Waiting for master to initialize", "Pod Phase", masterPod.Status.Phase)
		return reconcile.Result{}, err
	}

	//Create Deployment with Redis Slaves
	var slaves appsv1.Deployment
	slaves.Name = instance.Name + "-redis-slave"
	slaves.Namespace = instance.Namespace
	slaves.Spec.Selector.MatchLabels = map[string]string{
		"app":  instance.Name,
		"role": "slave",
		"tier": "backend",
	}
	_, err = ctrl.CreateOrUpdate(ctx, r.client, &slaves, func() error {
		modifyRedisSlavesForCR(instance, &slaves)
		return controllerutil.SetControllerReference(instance, &slaves, r.scheme)
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create Service for Redis Slaves
	var slavesSvc corev1.Service
	slavesSvc.Name = instance.Name + "-redis-slave-service"
	slavesSvc.Namespace = instance.Namespace
	_, err = ctrl.CreateOrUpdate(ctx, r.client, &slavesSvc, func() error {
		modifyRedisSlavesService(instance, &slavesSvc)
		return controllerutil.SetControllerReference(instance, &slavesSvc, r.scheme)
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Set Redis Active status
	if masterPod.Status.Phase == corev1.PodRunning && slaves.Status.ReadyReplicas == int32(instance.Spec.Size-1) {
		reqLogger.Info("Redis is ready")
		instance.Status.Ready = true
	} else {
		reqLogger.Info("Waiting for Redis to be ready")
		instance.Status.Ready = false
	}

	err = r.client.Update(context.TODO(), instance)

	return reconcile.Result{}, nil
}

// newRedisMasterForCR returns a redis master pod with the same name/namespace as the cr
func modifyRedisMasterForCR(cr *dysprozv1alpha1.Redis, master *corev1.Pod) {
	labels := map[string]string{
		"app":  cr.Name,
		"role": "master",
		"tier": "backend",
	}
	if master.ObjectMeta.Labels == nil {
		master.ObjectMeta.Labels = labels
	}
	if len(master.Spec.Containers) < 1 {
		master.Spec = corev1.PodSpec{
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
		}
	}
}

func modifyRedisSlavesForCR(cr *dysprozv1alpha1.Redis, slaves *appsv1.Deployment) {
	slavesReplicas := int32(cr.Spec.Size - 1)
	labels := map[string]string{
		"app":  cr.Name,
		"role": "slave",
		"tier": "backend",
	}
	if slaves.ObjectMeta.Labels == nil {
		slaves.ObjectMeta.Labels = labels
	}
	slaves.Spec.Replicas = &slavesReplicas
	templateSpec := &slaves.Spec.Template.Spec

	if len(templateSpec.Containers) == 0 {
		templateSpec.Containers = make([]corev1.Container, 1)
	}
	container := &templateSpec.Containers[0]
	container.Name = cr.Name + "-redis-slave"
	container.Image = cr.Spec.SlaveImage
	container.Ports = []corev1.ContainerPort{
		{
			ContainerPort: 6379,
		},
	}
	container.Env = []corev1.EnvVar{
		{
			Name:  "GET_HOSTS_FROM",
			Value: "env",
		},
		{
			Name:  "REDIS_MASTER_SERVICE_HOST",
			Value: cr.Name + "-redis-master-service",
		},
	}
}

func modifyRedisMasterService(cr *dysprozv1alpha1.Redis, masterSvc *corev1.Service) {
	labels := map[string]string{
		"app":  cr.Name,
		"role": "master",
		"tier": "backend",
	}
	if masterSvc.ObjectMeta.Labels == nil {
		masterSvc.ObjectMeta.Labels = labels
	}
	masterSvc.Spec.Ports =  []corev1.ServicePort{
			corev1.ServicePort{
				Port:       6379,
				TargetPort: intstr.IntOrString{IntVal: 6379},
			},
	}
	masterSvc.Spec.Selector = labels
}

func modifyRedisSlavesService(cr *dysprozv1alpha1.Redis, slaveSvc *corev1.Service) {
	labels := map[string]string{
		"app":  cr.Name,
		"role": "slave",
		"tier": "backend",
	}
	if slaveSvc.ObjectMeta.Labels == nil {
		slaveSvc.ObjectMeta.Labels = labels
	}
	slaveSvc.Spec.Ports = []corev1.ServicePort{
			corev1.ServicePort{
				Port: 6379,
			},
	}
	slaveSvc.Spec.Selector = labels
}
