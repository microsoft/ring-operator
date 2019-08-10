package ring_test

import (
	"context"
	"fmt"
	"github.com/microsoft/ring-operator/pkg/controller"
	"github.com/microsoft/ring-operator/pkg/controller/ring"
	"k8s.io/apimachinery/pkg/api/errors"
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

func createRing(name, branch, group string) *ringsv1alpha1.Ring {
	return &ringsv1alpha1.Ring{
		ObjectMeta: metav1.ObjectMeta{
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
						{
							Name: "SampleContainer",
							Ports: []corev1.ContainerPort{
								{
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
	//expectedPath := fmt.Sprintf("/%s/%s", selector["service"], selector["version"])
	//expectedRoute := fmt.Sprintf("PathPrefix(`%s`) && Headers(`group`, `%s`)", expectedPath, group)

	objs := []runtime.Object{
		controller.CreateRing(name, namespace, group, selector),
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
}

// TestReconcileProduction tests the special reconcile case of a production release
func TestReconcileProduction(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	// Create copy of instance with master branch
	namespace := "default"
	selector := map[string]string{"service": "query", "version": "v1", "branch": "master"}
	name := fmt.Sprintf("%s-%s-%s", selector["service"], selector["version"], selector["branch"])
	group := "*"
	//expectedPath := fmt.Sprintf("/%s/%s", selector["service"], selector["version"])
	//expectedRoute := fmt.Sprintf("PathPrefix(`%s`)", expectedPath)

	objs := []runtime.Object{
		controller.CreateRing(name, namespace, group, selector),
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
}

// TestReconcile tests a standard reconcile result for a non-master branch
func TestReconcileRing_Reconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	selector := map[string]string{"service": "query", "version": "v1", "branch": "canary"}
	name := fmt.Sprintf("%s-%s-%s", selector["service"], selector["version"], selector["branch"])
	canary := createRing(name, "canary", "canary")
	objs := []runtime.Object{canary}

	// Add Known CustomResourceDefinitions to the cluster scheme
	s := scheme.Scheme
	s.AddKnownTypes(ringsv1alpha1.SchemeGroupVersion, &ringsv1alpha1.Ring{})
	cl := fake.NewFakeClient(objs...)

	// Create a request for reconciliation
	r := &ring.ReconcileRing{Client: cl, Scheme: s}
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

	// Ensure Ring properly created and configured
	instance := &ringsv1alpha1.Ring{}
	err = cl.Get(context.TODO(), req.NamespacedName, instance)
	require.NoError(t, err)
	require.Equal(t, true, instance.Spec.Deploy)
	//require.NotEmpty(t, instance.)
	require.Equal(t, int32(80), instance.Spec.Routing.Ports[0].Port)

	masterSelector := map[string]string{"service": "query", "version": "v1", "branch": "master"}
	masterName := fmt.Sprintf("%s-%s-%s", masterSelector["service"], masterSelector["version"], masterSelector["branch"])
	masterInstance := createRing(masterName, "master", "*")
	err = cl.Create(context.TODO(), masterInstance)
	require.NoError(t, err)

	// Create a request for reconciliation
	r = &ring.ReconcileRing{Client: cl, Scheme: s}
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

	// Ensure Ring properly created and configured
	instance = &ringsv1alpha1.Ring{}
	err = cl.Get(context.TODO(), req.NamespacedName, instance)
	require.NoError(t, err)
	require.Equal(t, true, instance.Spec.Deploy)
	//require.NotEmpty(t, instance.Labels)
	require.Equal(t, int32(80), instance.Spec.Routing.Ports[0].Port)

	canary = createRing(name, "new", "newcanary")
	err = cl.Update(context.TODO(), canary)
	require.NoError(t, err)

	// Create a request for reconciliation
	r = &ring.ReconcileRing{Client: cl, Scheme: s}
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

	// Ensure Ring properly created and configured
	instance = &ringsv1alpha1.Ring{}
	err = cl.Get(context.TODO(), req.NamespacedName, instance)
	require.NoError(t, err)
	require.Equal(t, true, instance.Spec.Deploy)
	//require.NotEmpty(t, instance.Labels)
	require.Equal(t, int32(80), instance.Spec.Routing.Ports[0].Port)

	// Delete
	err = cl.Delete(context.TODO(), canary)
	require.NoError(t, err)

	// Create a request for reconciliation
	r = &ring.ReconcileRing{Client: cl, Scheme: s}
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

	// Ensure Ring properly created and configured
	instance = &ringsv1alpha1.Ring{}
	err = cl.Get(context.TODO(), req.NamespacedName, instance)
	require.Error(t, err)
	require.True(t, errors.IsNotFound(err))
}
