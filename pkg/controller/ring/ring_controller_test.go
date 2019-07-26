package ring_test

import (
	"context"
	"fmt"
	"github.com/microsoft/ring-operator/pkg/controller/ring"
	"testing"

	traefik "github.com/containous/traefik/pkg/provider/kubernetes/crd/traefik/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ringsv1alpha1 "github.com/microsoft/ring-operator/pkg/apis/rings/v1alpha1"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func createRing(name, namespace, group string, deploy bool, selector map[string]string) *ringsv1alpha1.Ring {
	return &ringsv1alpha1.Ring{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: ringsv1alpha1.RingSpec{
			Deploy: deploy,
			Routing: ringsv1alpha1.RingRouting{
				Group: ringsv1alpha1.RingGroup{
					Name: group,
				},
				Service: selector["service"],
				Version: selector["version"],
				Branch:  selector["branch"],
				Ports: []ringsv1alpha1.RingPort{
					ringsv1alpha1.RingPort{
						Name: "default",
						Port: 80,
					},
				},
			},
		},
	}
}

func createDeployment(name, namespace string, selector map[string]string) *appsv1.Deployment {
	var replicas int32 = 1
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selector,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name: "SampleContainer",
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									Name:          "SamplePort",
									Protocol:      "TCP",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}
}

// TestReconcile tests a standard reconcile result for a non-master branch
func TestReconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	// Setup the state of the cluster
	namespace := "default"
	selector := map[string]string{"service": "query", "version": "v1", "branch": "canary"}
	name := fmt.Sprintf("%s-%s-%s", selector["service"], selector["version"], selector["branch"])
	group := "canary"
	expectedPath := fmt.Sprintf("/%s/%s", selector["service"], selector["version"])
	expectedRoute := fmt.Sprintf("PathPrefix(`%s`) && Headers(`group`, `%s`)", expectedPath, group)

	objs := []runtime.Object{
		createRing(name, namespace, group, true, selector),
		createDeployment(name, namespace, selector),
	}

	// Add Known CustomResourceDefinitions to the cluster scheme
	s := scheme.Scheme
	s.AddKnownTypes(ringsv1alpha1.SchemeGroupVersion, &ringsv1alpha1.Ring{})
	s.AddKnownTypes(traefik.SchemeGroupVersion, &traefik.IngressRoute{})
	s.AddKnownTypes(traefik.SchemeGroupVersion, &traefik.Middleware{})
	cl := fake.NewFakeClient(objs...)

	// Create a request for reconciliation
	r := &ring.ReconcileRing{Client: cl, Scheme: s}
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}

	// Reconcile request
	res, err := r.Reconcile(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Ensure Middleware properly created and configured
	m := &traefik.Middleware{}
	err = cl.Get(context.TODO(), types.NamespacedName{Namespace: req.Namespace, Name: fmt.Sprintf("%s-stripprefix", name)}, m)
	require.NoError(t, err)
	require.NotNil(t, m.Spec.StripPrefix)
	require.NotEmpty(t, m.Spec.StripPrefix.Prefixes)
	require.Equal(t, expectedPath, m.Spec.StripPrefix.Prefixes[0])

	// Ensure Service properly created and configured
	svc := &corev1.Service{}
	err = cl.Get(context.TODO(), req.NamespacedName, svc)
	require.NoError(t, err)
	require.Equal(t, selector, svc.Spec.Selector)
	require.NotEmpty(t, svc.Spec.Ports)
	require.Equal(t, int32(80), svc.Spec.Ports[0].Port)

	// Ensure IngressRoute properly created and configured
	ing := &traefik.IngressRoute{}
	err = cl.Get(context.TODO(), req.NamespacedName, ing)
	require.NoError(t, err)
	require.NotEmpty(t, ing.Spec.EntryPoints)
	require.Equal(t, []string{"http", "https", "internal"}, ing.Spec.EntryPoints)
	require.NotEmpty(t, ing.Spec.Routes)
	require.Equal(t, expectedRoute, ing.Spec.Routes[0].Match)
	require.NotEmpty(t, ing.Spec.Routes[0].Middlewares)
	require.Equal(t, fmt.Sprintf("%s-stripprefix", name), ing.Spec.Routes[0].Middlewares[0].Name)
}

// TestReconcileProduction tests the special reconcile case of a production release
func TestReconcileProduction(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	// Create copy of instance with master branch
	namespace := "default"
	selector := map[string]string{"service": "query", "version": "v1", "branch": "master"}
	name := fmt.Sprintf("%s-%s-%s", selector["service"], selector["version"], selector["branch"])
	group := "*"
	expectedPath := fmt.Sprintf("/%s/%s", selector["service"], selector["version"])
	expectedRoute := fmt.Sprintf("PathPrefix(`%s`)", expectedPath)

	objs := []runtime.Object{
		createRing(name, namespace, group, true, selector),
		createDeployment(name, namespace, selector),
	}

	// Add Known CustomResourceDefinitions to the cluster scheme
	s := scheme.Scheme
	s.AddKnownTypes(ringsv1alpha1.SchemeGroupVersion, &ringsv1alpha1.Ring{})
	s.AddKnownTypes(traefik.SchemeGroupVersion, &traefik.IngressRoute{})
	s.AddKnownTypes(traefik.SchemeGroupVersion, &traefik.Middleware{})
	cl := fake.NewFakeClient(objs...)

	// Create a request for reconciliation
	r := &ring.ReconcileRing{Client: cl, Scheme: s}
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}

	// Reconcile request
	res, err := r.Reconcile(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Ensure Middleware properly created and configured
	m := &traefik.Middleware{}
	err = cl.Get(context.TODO(), types.NamespacedName{Namespace: req.Namespace, Name: fmt.Sprintf("%s-stripprefix", name)}, m)
	require.NoError(t, err)
	require.NotNil(t, m.Spec.StripPrefix)
	require.NotEmpty(t, m.Spec.StripPrefix.Prefixes)
	require.Equal(t, expectedPath, m.Spec.StripPrefix.Prefixes[0])

	// Ensure Service properly created and configured
	svc := &corev1.Service{}
	err = cl.Get(context.TODO(), req.NamespacedName, svc)
	require.NoError(t, err)
	require.Equal(t, selector, svc.Spec.Selector)
	require.NotEmpty(t, svc.Spec.Ports)
	require.Equal(t, int32(80), svc.Spec.Ports[0].Port)

	// Ensure IngressRoute properly created and configured
	ing := &traefik.IngressRoute{}
	err = cl.Get(context.TODO(), req.NamespacedName, ing)
	require.NoError(t, err)
	require.NotEmpty(t, ing.Spec.EntryPoints)
	require.Equal(t, []string{"http", "https", "internal"}, ing.Spec.EntryPoints)
	require.NotEmpty(t, ing.Spec.Routes)
	require.Equal(t, expectedRoute, ing.Spec.Routes[0].Match)
	require.NotEmpty(t, ing.Spec.Routes[0].Middlewares)
	require.Equal(t, fmt.Sprintf("%s-stripprefix", name), ing.Spec.Routes[0].Middlewares[0].Name)
}

func TestReconcileDeployFalse(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	// Create copy of instance with deploy: false
	namespace := "default"
	selector := map[string]string{"service": "query", "version": "v1", "branch": "canary"}
	name := fmt.Sprintf("%s-%s-%s", selector["service"], selector["version"], selector["branch"])
	group := "canary"

	objs := []runtime.Object{
		createRing(name, namespace, group, false, selector),
		createDeployment(name, namespace, selector),
	}

	// Add Known CustomResourceDefinitions to the cluster scheme
	s := scheme.Scheme
	s.AddKnownTypes(ringsv1alpha1.SchemeGroupVersion, &ringsv1alpha1.Ring{})
	s.AddKnownTypes(traefik.SchemeGroupVersion, &traefik.IngressRoute{})
	cl := fake.NewFakeClient(objs...)

	// Create a request for reconciliation
	r := &ring.ReconcileRing{Client: cl, Scheme: s}
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}

	// Reconcile request
	res, err := r.Reconcile(req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.False(t, res.Requeue)
}
