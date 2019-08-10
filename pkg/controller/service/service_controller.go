package service

import (
	"context"
	"github.com/go-logr/logr"
	ringsv1alpha1 "github.com/microsoft/ring-operator/pkg/apis/rings/v1alpha1"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_service")

// Add creates a new Service Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileService{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	debugLog := log.V(int(zapcore.DebugLevel))
	debugLog.Info("Creating a new Ring controller")
	c, err := controller.New("service-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	debugLog.Info("Adding watch for Ring resource")
	err = c.Watch(&source.Kind{Type: &ringsv1alpha1.Ring{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		log.Error(err, "Could not watch resource Ring")
		return err
	}

	debugLog.Info("Adding watch for child Service")
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ringsv1alpha1.Ring{},
	})
	if err != nil {
		log.Error(err, "Could not watch child resource Service")
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileService{}

// ReconcileService reconciles a Service object
type ReconcileService struct {
	// This Client, initialized using mgr.Client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	Client client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
	debug  logr.InfoLogger
}

// Reconcile reads that state of the cluster for a Service object and makes changes based on the state read
// and what is in the Service.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
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
	if !instance.Spec.Deploy {
		r.debug.Info("Ring deploy is set to false - don't requeue the request")
		return reconcile.Result{Requeue: false}, nil
	}

	r.debug.Info("Ensure Service exists")
	if _, err := r.createOrUpdateService(instance); err != nil {
		r.logger.Error(err, "Could not create or update service")
		return reconcile.Result{}, err
	}

	r.logger.Info("Reconciliation finished")
	return reconcile.Result{}, err
}

// createOrUpdateService ensures the Service exists with the up to date information in the Ring instance
// It returns created or updated Service and any error
func (r *ReconcileService) createOrUpdateService(cr *ringsv1alpha1.Ring) (*corev1.Service, error) {
	r.logger.Info("createOrUpdateService")

	svcFound := &corev1.Service{}
	r.logger.Info("Finding Service")
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, svcFound)
	if err != nil && errors.IsNotFound(err) {
		svc := r.newServiceForCR(cr)

		r.debug.Info("Setting Ring as owner of service")
		if err := controllerutil.SetControllerReference(cr, svc, r.Scheme); err != nil {
			r.logger.Error(err, "Could not set Ring as owner of Service")
			return nil, err
		}

		r.logger.Info("Creating a new Service")
		if err = r.Client.Create(context.TODO(), svc); err != nil {
			r.logger.Error(err, "Could not create Service")
			return nil, err
		}

		return svc, nil
	} else if err != nil {
		r.logger.Error(err, "Could not get existing Service")
		return nil, err
	} else {
		r.logger.Info("Updating service")
		svc := r.updateServiceForCR(svcFound, cr)
		if err = r.Client.Update(context.TODO(), svc); err != nil {
			r.logger.Info("Could not update service")
			return nil, err
		}
		return svc, nil
	}
}

// getServicePorts returns the Service port representation of the ports in the Ring CRD
func getServicePorts(routing *ringsv1alpha1.RingRouting) []corev1.ServicePort {
	ports := make([]corev1.ServicePort, len(routing.Ports))
	for i, port := range routing.Ports {
		ports[i] = corev1.ServicePort{
			Name:       port.Name,
			Port:       port.Port,
			Protocol:   port.Protocol,
			TargetPort: port.TargetPort,
		}
	}
	return ports
}

// createService will create the service for accessing the target Pods by name
func (r *ReconcileService) newServiceForCR(cr *ringsv1alpha1.Ring) *corev1.Service {
	routing := cr.Spec.Routing
	selector := map[string]string{
		"service": routing.Service,
		"version": routing.Version,
		"branch":  routing.Branch,
	}

	ports := getServicePorts(&routing)
	objMeta := metav1.ObjectMeta{
		Name:      cr.Name,
		Namespace: cr.Namespace,
		Labels:    cr.ObjectMeta.Labels,
	}

	return &corev1.Service{
		ObjectMeta: objMeta,
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: selector,
		},
	}
}

func (r *ReconcileService) updateServiceForCR(svc *corev1.Service, cr *ringsv1alpha1.Ring) *corev1.Service {
	newSvc := svc.DeepCopy()

	routing := cr.Spec.Routing
	selector := map[string]string{
		"service": routing.Service,
		"version": routing.Version,
		"branch":  routing.Branch,
	}

	ports := getServicePorts(&routing)
	newSvc.Labels = cr.ObjectMeta.Labels
	newSvc.Spec.Ports = ports
	newSvc.Spec.Selector = selector
	return newSvc
}
