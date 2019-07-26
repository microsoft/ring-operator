package ring

import (
	"context"
	ringsv1alpha1 "github.com/microsoft/ring-operator/pkg/apis/rings/v1alpha1"
)

const ringFinalizer = "finalizer.rings.microsoft.com"

// finalizeRing runs the steps which happen when the ring is going to be destroyed
// these steps include deleting the AAD Group that backs the ring
func (r *ReconcileRing) finalizeRing(cr *ringsv1alpha1.Ring) error {
	r.logger.Info("Finalizing ring")

	// TODO - Add check if this is the last ring for that group
	// Only the delete the group if no other rings are using it

	//return r.deleteADGroup(cr)
	return nil
}

func (r *ReconcileRing) addFinalizer(cr *ringsv1alpha1.Ring) error {
	r.logger.Info("Adding Finalizer for Ring")
	cr.SetFinalizers(append(cr.GetFinalizers(), ringFinalizer))

	err := r.Client.Update(context.TODO(), cr)
	if err != nil {
		r.logger.Error(err, "Failed to update Ring with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
