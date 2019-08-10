package controller

import (
	ringsv1alpha1 "github.com/microsoft/ring-operator/pkg/apis/rings/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateRing(name, namespace, group string, selector map[string]string) *ringsv1alpha1.Ring {
	return &ringsv1alpha1.Ring{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: ringsv1alpha1.RingSpec{
			Deploy: true,
			Routing: ringsv1alpha1.RingRouting{
				Group: ringsv1alpha1.RingGroup{
					Name: group,
				},
				Service: selector["service"],
				Version: selector["version"],
				Branch:  selector["branch"],
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

