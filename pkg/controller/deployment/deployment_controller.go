package deployment

import (
	"context"

	"github.com/wave-k8s/wave/pkg/core"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new Deployment Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDeployment{
		scheme:  mgr.GetScheme(),
		handler: core.NewHandler(mgr.GetClient(), mgr.GetEventRecorderFor("wave")),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("deployment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Deployment
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch ConfigMaps owned by a Deployment
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: false,
		OwnerType:    &appsv1.Deployment{},
	})
	if err != nil {
		return err
	}

	// Watch Secrets owned by a Deployment
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: false,
		OwnerType:    &appsv1.Deployment{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileDeployment{}

// ReconcileDeployment reconciles a Deployment object
type ReconcileDeployment struct {
	scheme  *runtime.Scheme
	handler *core.Handler
}

// Reconcile reads that state of the cluster for a Deployment object and
// updates its PodSpec based on mounted configuration
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=secrets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=events,verbs=create;update;patch
func (r *ReconcileDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Deployment instance
	instance := &appsv1.Deployment{}
	err := r.handler.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	return r.handler.HandleDeployment(instance)
}
