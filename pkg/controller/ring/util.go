package ring

import (
	"fmt"
	"os"
	ringsv1alpha1 "ring-operator/pkg/apis/rings/v1alpha1"

	traefikcfg "github.com/containous/traefik/pkg/config"
	traefik "github.com/containous/traefik/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ratelimitMiddlewareName   = "%s-ratelimit"
	stripPrefixMiddlewareName = "%s-stripprefix"
)

// createIngressRoute will create the Traefik IngressRoute resource to handle routing from external to the service
func (r *ReconcileRing) newIngressRouteForCR(cr *ringsv1alpha1.Ring) *traefik.IngressRoute {
	r.logger.Info("Creating Ingress Route", "IngressRoute.Namespace", cr.Namespace, "IngressRoute.Name", cr.Name)

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

func (r *ReconcileRing) updateIngressRouteForCR(ing *traefik.IngressRoute, cr *ringsv1alpha1.Ring) *traefik.IngressRoute {
	r.logger.Info("Updating Ingress Route", "IngressRoute.Namespace", cr.Namespace, "IngressRoute.Name", cr.Name)

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

// createStripPrefixMiddlewareRef returns a middleware reference to the stripPrefix
// middleware associated with this ring
func createStripPrefixMiddlewareRef(name, namespace string) traefik.MiddlewareRef {
	return traefik.MiddlewareRef{
		Name:      fmt.Sprintf(stripPrefixMiddlewareName, name),
		Namespace: namespace,
	}
}

// createRateLimitMiddlewareRef returns a middleware reference to the rateLimit
// middleware associated with this ring
func createRateLimitMiddlewareRef(name, namespace string) traefik.MiddlewareRef {
	return traefik.MiddlewareRef{
		Name:      fmt.Sprintf(ratelimitMiddlewareName, name),
		Namespace: namespace,
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
func (r *ReconcileRing) newServiceForCR(cr *ringsv1alpha1.Ring) *corev1.Service {
	r.logger.Info("Creating Service", "Service.Namespace", cr.Namespace, "Service.Name", cr.Name)

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

func (r *ReconcileRing) updateServiceForCR(svc *corev1.Service, cr *ringsv1alpha1.Ring) *corev1.Service {
	r.logger.Info("Updating Service", "Service.Namespace", cr.Namespace, "Service.Name", cr.Name)
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

// newStripPrefixForCR creates a new Traefik Middleware object (not yet created) representing
// a StripPrefix to remove the path from the request so that the service only sees the path it expects
// out of the cluster (eg: /hello-world/v1/home.html -> /home.html)
func (r *ReconcileRing) newStripPrefixForCR(cr *ringsv1alpha1.Ring) *traefik.Middleware {
	r.logger.Info("Creating StripPrefix", "StripPrefix.Namespace", cr.Namespace, "StripPrefix.Name", cr.Name)

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

func (r *ReconcileRing) updateStripPrefixForCR(sp *traefik.Middleware, cr *ringsv1alpha1.Ring) *traefik.Middleware {
	r.logger.Info("Updating StripPrefix", "StripPrefix.Namespace", cr.Namespace, "StripPrefix.Name", cr.Name)
	newSp := sp.DeepCopy()

	routing := cr.Spec.Routing
	path := fmt.Sprintf("/%s/%s", routing.Service, routing.Version)

	newSp.Labels = cr.ObjectMeta.Labels
	newSp.Spec.StripPrefix.Prefixes = []string{path}
	return newSp
}
