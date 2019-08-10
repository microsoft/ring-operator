package ingressroute

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"os"

	traefikcfg "github.com/containous/traefik/pkg/config"
	traefik "github.com/containous/traefik/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	ringsv1alpha1 "github.com/microsoft/ring-operator/pkg/apis/rings/v1alpha1"
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

const (
	//ratelimitMiddlewareName   = "%s-ratelimit"
	stripPrefixMiddlewareName = "%s-stripprefix"
)

var log = logf.Log.WithName("controller_ingressroute")

// Add creates a new IngressRoute Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileIngressRoute{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	debugLog := log.V(int(zapcore.DebugLevel))
	debugLog.Info("Creating a new IngressRoute controller")
	c, err := controller.New("ingressroute-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	debugLog.Info("Adding Traefik scheme to controller")
	if err := traefik.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "Could not add traefik scheme")
		return err
	}

	debugLog.Info("Adding watch for Ring resource")
	err = c.Watch(&source.Kind{Type: &ringsv1alpha1.Ring{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		log.Error(err, "Could not watch resource Ring")
		return err
	}

	debugLog.Info("Adding watch for child Traefik Middleware")
	err = c.Watch(&source.Kind{Type: &traefik.Middleware{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ringsv1alpha1.Ring{},
	})
	if err != nil {
		log.Error(err, "Could not watch child resource Middleware")
		return err
	}

	debugLog.Info("Adding watch for child Traefik IngressRoute")
	err = c.Watch(&source.Kind{Type: &traefik.IngressRoute{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ringsv1alpha1.Ring{},
	})
	if err != nil {
		log.Error(err, "Could not watch child resource IngressRoute")
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileIngressRoute implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileIngressRoute{}

// ReconcileIngressRoute reconciles a IngressRoute object
type ReconcileIngressRoute struct {
	// This Client, initialized using mgr.Client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	Client client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
	debug  logr.InfoLogger
}

// Reconcile reads that state of the cluster for a IngressRoute object and makes changes based on the state read
// and what is in the IngressRoute.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIngressRoute) Reconcile(request reconcile.Request) (reconcile.Result, error) {
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
	r.debug.Info("Ensure StripPrefix exists")
	if _, err := r.createOrUpdateStripPrefix(instance); err != nil {
		r.logger.Error(err, "Could not create or update stripPrefix")
		return reconcile.Result{}, err
	}

	r.debug.Info("Ensure IngressRoute exists")
	if _, err := r.createOrUpdateIngressRoute(instance); err != nil {
		r.logger.Error(err, "Could not create or update ingress route")
		return reconcile.Result{}, err
	}

	r.logger.Info("Reconciliation finished")
	return reconcile.Result{}, err
}

// createOrUpdateIngressRoute ensures the IngressRoute exists with the up to date information in the Ring instance
// It returns created or updated IngressRoute and any error
func (r *ReconcileIngressRoute) createOrUpdateIngressRoute(cr *ringsv1alpha1.Ring) (*traefik.IngressRoute, error) {
	r.logger.Info("createOrUpdateIngressRoute")

	ingFound := &traefik.IngressRoute{}
	r.logger.Info("Finding IngressRoute")
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, ingFound)
	if err != nil && errors.IsNotFound(err) {
		r.logger.Info("Creating Ingress Route")
		ing := newIngressRouteForCR(cr)

		r.debug.Info("Setting Ring as owner of IngressRoute")
		if err := controllerutil.SetControllerReference(cr, ing, r.Scheme); err != nil {
			r.logger.Error(err, "Could not set Ring as owner of IngressRoute")
			return nil, err
		}

		r.logger.Info("Creating a new IngressRoute")
		if err = r.Client.Create(context.TODO(), ing); err != nil {
			r.logger.Error(err, "Could not create IngressRoute")
			return nil, err
		}
		return ing, nil
	} else if err != nil {
		r.logger.Error(err, "Could not get existing IngressRoute")
		return nil, err
	} else {
		r.logger.Info("Updating IngressRoute")
		ing := updateIngressRouteForCR(ingFound, cr)
		if err = r.Client.Update(context.TODO(), ing); err != nil {
			r.logger.Info("Could not update IngressRoute")
			return nil, err
		}
		return ing, nil
	}
}

// createOrUpdateStripPrefix ensures the StripPrefix Middleware exists with the up to date information in the Ring instance
// It returns created or updated StripPrefix Middleware and any error
func (r *ReconcileIngressRoute) createOrUpdateStripPrefix(cr *ringsv1alpha1.Ring) (*traefik.Middleware, error) {
	r.debug.Info("createOrUpdateStripPrefix")

	r.logger.Info("Finding StripPrefix")
	mFound := &traefik.Middleware{}
	mName := fmt.Sprintf("%s-stripprefix", cr.Name)

	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: mName, Namespace: cr.Namespace}, mFound)
	if err != nil && errors.IsNotFound(err) {
		m := newStripPrefixForCR(cr)

		r.debug.Info("Setting Ring as owner of StripPrefix")
		if err := controllerutil.SetControllerReference(cr, m, r.Scheme); err != nil {
			r.logger.Error(err, "Could not set Ring as owner of Middleware")
			return nil, err
		}

		r.logger.Info("Creating a new StripPrefix")
		if err = r.Client.Create(context.TODO(), m); err != nil {
			r.logger.Error(err, "Could not create StripPrefix")
			return nil, err
		}

		return m, nil
	} else if err != nil {
		r.logger.Error(err, "Could not get existing StripPrefix")
		return nil, err
	} else {
		r.logger.Info("Updating StripPrefix")
		m := updateStripPrefixForCR(mFound, cr)
		if err = r.Client.Update(context.TODO(), m); err != nil {
			r.logger.Info("Could not update StripPrefix")
			return nil, err
		}
		return m, nil
	}
}

// createIngressRoute will create the Traefik IngressRoute resource to handle routing from external to the service
func newIngressRouteForCR(cr *ringsv1alpha1.Ring) *traefik.IngressRoute {
	// Use entrypoints set for Traefik
	entryPoints := []string{
		"http",
		"https",
		"internal",
	}

	// Create match rule from routing descriptor
	routing := cr.Spec.Routing
	match := createMatchRule(&routing)

	// Get service ports
	serviceName := fmt.Sprintf("%s-%s-%s", routing.Service, routing.Version, routing.Branch)
	ports := getTraefikServices(serviceName, &routing)

	middlewareRefs := []traefik.MiddlewareRef{
		// createRateLimitMiddlewareRef(serviceName, cr.Namespace),
		createStripPrefixMiddlewareRef(serviceName, cr.Namespace),
	}

	objMeta := metav1.ObjectMeta{
		Name:      cr.Name,
		Namespace: cr.Namespace,
		Labels:    cr.ObjectMeta.Labels,
	}

	return &traefik.IngressRoute{
		ObjectMeta: objMeta,
		Spec: traefik.IngressRouteSpec{
			EntryPoints: entryPoints,
			Routes: []traefik.Route{
				{
					Match:       match,
					Kind:        "Rule",
					Services:    ports,
					Middlewares: middlewareRefs,
				},
			},
		},
	}
}

func updateIngressRouteForCR(ing *traefik.IngressRoute, cr *ringsv1alpha1.Ring) *traefik.IngressRoute {
	newIng := ing.DeepCopy()

	// Create match rule from routing descriptor
	routing := cr.Spec.Routing
	match := createMatchRule(&routing)

	// Get service ports
	ports := getTraefikServices(cr.Name, &routing)

	middlewareRefs := []traefik.MiddlewareRef{
		// createRateLimitMiddlewareRef(serviceName, cr.Namespace),
		createStripPrefixMiddlewareRef(cr.Name, cr.Namespace),
	}

	newIng.Labels = cr.ObjectMeta.Labels
	newIng.Spec.Routes = []traefik.Route{
		{
			Match:       match,
			Kind:        "Rule",
			Services:    ports,
			Middlewares: middlewareRefs,
		},
	}
	return newIng
}

// createStripPrefixMiddlewareRef returns a middleware reference to the stripPrefix
// middleware associated with this ring
func createStripPrefixMiddlewareRef(name, namespace string) traefik.MiddlewareRef {
	return traefik.MiddlewareRef{
		Name:      fmt.Sprintf(stripPrefixMiddlewareName, name),
		Namespace: namespace,
	}
}

// getTraefikServices returns a mapping from the ring port definition into the Traefik service definition
// as required for the IngressRoute
func getTraefikServices(serviceName string, routing *ringsv1alpha1.RingRouting) []traefik.Service {
	ports := routing.Ports
	tPorts := make([]traefik.Service, len(ports))
	for i, port := range ports {
		tPorts[i] = traefik.Service{
			Name: serviceName,
			Port: port.Port,
		}
	}
	return tPorts
}

// createMatchRule will generate a routing rule for the ring
// it handles special cases such as production ring
func createMatchRule(routing *ringsv1alpha1.RingRouting) string {
	// Handle production
	if routing.Group.Name == "*" {
		return fmt.Sprintf("PathPrefix(`/%s/%s`)", routing.Service, routing.Version)
	}

	var (
		routingKey     = "group"
		ringRoutingKey = ""
	)

	ringRoutingKey = os.Getenv("RING_ROUTING_KEY")
	if ringRoutingKey != "" {
		routingKey = ringRoutingKey
	}

	return fmt.Sprintf("PathPrefix(`/%s/%s`) && Headers(`%s`, `%s`)", routing.Service, routing.Version, routingKey, routing.Group.Name)
}

// newStripPrefixForCR creates a new Traefik Middleware object (not yet created) representing
// a StripPrefix to remove the path from the request so that the service only sees the path it expects
// out of the cluster (eg: /hello-world/v1/home.html -> /home.html)
func newStripPrefixForCR(cr *ringsv1alpha1.Ring) *traefik.Middleware {
	routing := cr.Spec.Routing
	path := fmt.Sprintf("/%s/%s", routing.Service, routing.Version)
	objMeta := metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-stripprefix", cr.Name),
		Namespace: cr.Namespace,
		Labels:    cr.ObjectMeta.Labels,
	}

	return &traefik.Middleware{
		ObjectMeta: objMeta,
		Spec: traefikcfg.Middleware{
			StripPrefix: &traefikcfg.StripPrefix{Prefixes: []string{path}},
		},
	}
}

func updateStripPrefixForCR(sp *traefik.Middleware, cr *ringsv1alpha1.Ring) *traefik.Middleware {
	newSp := sp.DeepCopy()

	routing := cr.Spec.Routing
	path := fmt.Sprintf("/%s/%s", routing.Service, routing.Version)

	newSp.Labels = cr.ObjectMeta.Labels
	newSp.Spec.StripPrefix.Prefixes = []string{path}
	return newSp
}
