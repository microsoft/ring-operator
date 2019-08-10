package service_test

import (
	"context"
	"fmt"
	"github.com/microsoft/ring-operator/pkg/controller/service"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	traefik "github.com/containous/traefik/pkg/provider/kubernetes/crd/traefik/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ringsv1alpha1 "github.com/microsoft/ring-operator/pkg/apis/rings/v1alpha1"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func createRing(name, branch, group string) *ringsv1alpha1.Ring {
	return &ringsv1alpha1.Ring{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: ringsv1alpha1.RingSpec{
			Deploy: true,
			Routing: ringsv1alpha1.RingRouting{
				Group:   ringsv1alpha1.RingGroup{Name: group},
				Service: "query",
				Version: "v1",
				Branch:  branch,
				Ports: []ringsv1alpha1.RingPort{
					{
						Name: "default",
						Port: 80,
					},
				},
			},
		},
	}
}

// TestReconcile tests a standard reconcile result for a non-master branch
func TestReconcileService_Reconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	selector := map[string]string{"service": "query", "version": "v1", "branch": "canary"}
	name := fmt.Sprintf("%s-%s-%s", selector["service"], selector["version"], selector["branch"])
	canary := createRing(name, "canary", "canary")
	objs := []runtime.Object{canary}

	// Add Known CustomResourceDefinitions to the cluster scheme
	s := scheme.Scheme
	s.AddKnownTypes(ringsv1alpha1.SchemeGroupVersion, &ringsv1alpha1.Ring{})
	s.AddKnownTypes(traefik.SchemeGroupVersion, &traefik.IngressRoute{})
	s.AddKnownTypes(traefik.SchemeGroupVersion, &traefik.Middleware{})
	cl := fake.NewFakeClient(objs...)

	// Create a request for reconciliation
	r := &service.ReconcileService{Client: cl, Scheme: s}
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: "default",
		},
	}

	// Reconcile request
	res, err := r.Reconcile(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Ensure Service properly created and configured
	svc := &corev1.Service{}
	err = cl.Get(context.TODO(), req.NamespacedName, svc)
	require.NoError(t, err)
	require.Equal(t, selector, svc.Spec.Selector)
	require.NotEmpty(t, svc.Spec.Ports)
	require.Equal(t, int32(80), svc.Spec.Ports[0].Port)

	masterSelector := map[string]string{"service": "query", "version": "v1", "branch": "master"}
	masterName := fmt.Sprintf("%s-%s-%s", masterSelector["service"], masterSelector["version"], masterSelector["branch"])
	masterInstance := createRing(masterName, "master", "*")
	err = cl.Create(context.TODO(), masterInstance)
	require.NoError(t, err)

	// Create a request for reconciliation
	r = &service.ReconcileService{Client: cl, Scheme: s}
	req = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      masterName,
			Namespace: "default",
		},
	}

	// Reconcile request
	res, err = r.Reconcile(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Ensure Service properly created and configured
	svc = &corev1.Service{}
	err = cl.Get(context.TODO(), req.NamespacedName, svc)
	require.NoError(t, err)
	require.Equal(t, masterSelector, svc.Spec.Selector)
	require.NotEmpty(t, svc.Spec.Ports)
	require.Equal(t, int32(80), svc.Spec.Ports[0].Port)

	canary = createRing(name, "new", "newcanary")
	err = cl.Update(context.TODO(), canary)
	require.NoError(t, err)

	// Create a request for reconciliation
	r = &service.ReconcileService{Client: cl, Scheme: s}
	req = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: "default",
		},
	}

	// Reconcile request
	res, err = r.Reconcile(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Ensure Service properly created and configured
	svc = &corev1.Service{}
	err = cl.Get(context.TODO(), req.NamespacedName, svc)
	require.NoError(t, err)
	canarySelector := map[string]string{"service": "query", "version": "v1", "branch": "new"}
	require.Equal(t, canarySelector, svc.Spec.Selector)
	require.NotEmpty(t, svc.Spec.Ports)
	require.Equal(t, int32(80), svc.Spec.Ports[0].Port)

	// Delete
	err = cl.Delete(context.TODO(), canary)
	require.NoError(t, err)

	// Create a request for reconciliation
	r = &service.ReconcileService{Client: cl, Scheme: s}
	req = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: "default",
		},
	}

	// Reconcile request
	res, err = r.Reconcile(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Ensure Service properly created and configured
	svc = &corev1.Service{}
	err = cl.Get(context.TODO(), req.NamespacedName, svc)
	require.NoError(t, err)
	canarySelector = map[string]string{"service": "query", "version": "v1", "branch": "new"}
	require.Equal(t, canarySelector, svc.Spec.Selector)
	require.NotEmpty(t, svc.Spec.Ports)
	require.Equal(t, int32(80), svc.Spec.Ports[0].Port)
}
