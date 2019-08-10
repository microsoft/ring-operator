package ring

import (
    "context"
    "github.com/go-logr/logr"
    "go.uber.org/zap/zapcore"
    "os"
    "strings"

    ringsv1alpha1 "github.com/microsoft/ring-operator/pkg/apis/rings/v1alpha1"

    "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/controller"
    "sigs.k8s.io/controller-runtime/pkg/handler"
    "sigs.k8s.io/controller-runtime/pkg/manager"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"
    logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
    "sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_ring")

// Add creates a new Ring Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRing{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	debugLog := log.V(int(zapcore.DebugLevel))
	debugLog.Info("Creating a new Ring controller")
	c, err := controller.New("ring-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	debugLog.Info("Adding watch for Ring resource")
	err = c.Watch(&source.Kind{Type: &ringsv1alpha1.Ring{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		log.Error(err, "Could not watch resource Ring")
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRing implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRing{}

// ReconcileRing reconciles a Ring object
type ReconcileRing struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
	debug  logr.InfoLogger
}

// Reconcile reads that state of the cluster for a Ring object and makes changes based on the state read
// and what is in the Ring.Spec
// Steps:
// 1. Create Middleware specific to this Ring
//		a. StripPrefix
// 2. Create Service to link Deployment
// 3. Create IngressRoute to link Service
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRing) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.logger = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	r.debug = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name).V(int(zapcore.DebugLevel))

	r.debug.Info("Starting Ring reconciliation")
	instance := &ringsv1alpha1.Ring{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		r.logger.Error(err, "Could not get the Ring instance")
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.logger.Error(err, "Ring instance not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.logger.Error(err, "Could not get the Ring instance - Requeue the request")
		return reconcile.Result{}, nil
	}

	r.debug.Info("Found the Ring instance")
	r.debug.Info("Setting finalizer to run when deletion happens")
	if err := r.handleDeletion(instance); err != nil {
		r.logger.Error(err, "Error handling deletion finalizer")
		return reconcile.Result{}, err
	}

	useADGroups := os.Getenv("AZURE_AD_ENABLED")
	if strings.ToLower(useADGroups) == "true" {
		r.debug.Info("Checking if AD group already exists")
		if adGroupExists, err := r.adGroupExists(instance); err != nil {
			r.logger.Error(err, "Could not check if AD Group Exists")
			return reconcile.Result{}, err
		} else if !adGroupExists {
			r.debug.Info("AD Group does not exist")
			if err := r.createADGroup(instance); err != nil {
				r.logger.Error(err, "Could not create AD group")
				return reconcile.Result{}, err
			}
		}
	}

	r.logger.Info("Reconciliation finished")
	return reconcile.Result{}, err
}

// handleDeletion sets up this ring for deletion
// It checks if the ring is marked for deletion
// If it's marked for deletion then it should clean up all off-cluster resources (eg: AAD Groups)
// If it isn't marked for deletion then it should ensure that the finalizer is set on the instance
func (r *ReconcileRing) handleDeletion(cr *ringsv1alpha1.Ring) error {
	r.debug.Info("Starting handling deletion")
	r.debug.Info("Check if Ring is marked for deletion")
	if isRingMarkedForDeletion := cr.GetDeletionTimestamp(); isRingMarkedForDeletion != nil {

		r.debug.Info("Check if finalizer has run")
		if contains(cr.GetFinalizers(), ringFinalizer) {
			// TODO - Cleanup and remove off-cluster resources

			useADGroup := os.Getenv("AZURE_AD_ENABLED")
			if strings.ToLower(useADGroup) == "true" {
				r.logger.Info("Deleting AAD Group")
				if err := r.finalizeRing(cr); err != nil {
					r.logger.Error(err, "Could not finalize the ring")
					return err
				}
			}

			r.debug.Info("Removing finalizer from Ring resource to allow deletion")
			cr.SetFinalizers(remove(cr.GetFinalizers(), ringFinalizer))

			r.logger.Info("Updating the ring to remove finalizer")
			err := r.Client.Update(context.TODO(), cr)
			if err != nil {
				r.logger.Error(err, "Could not update the Ring to remove finalizer")
				return err
			}
		}

		r.debug.Info("Finished handleDeletion")
		return nil
	}

	r.debug.Info("Checking if finalizer exists on Ring resource")
	if !contains(cr.GetFinalizers(), ringFinalizer) {
		r.debug.Info("Adding finalizer to the Ring")

		//if cr.Spec.Routing.Group.InitialUsers == nil {
		//    cr.Spec.Routing.Group.InitialUsers = []string{}
		//}

		if err := r.addFinalizer(cr); err != nil {
			r.logger.Error(err, "Could not add finalizer to the Ring")
			return err
		}
	}

	return nil
}
