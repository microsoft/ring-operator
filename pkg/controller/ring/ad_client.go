package ring

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"io/ioutil"
	"os"
	ringsv1alpha1 "ring-operator/pkg/apis/rings/v1alpha1"
)

// CreateADGroup will create the AAD group in Azure
func (r *ReconcileRing) createADGroup(cr *ringsv1alpha1.Ring) error {
	tenantId := os.Getenv("AZURE_TENANT_ID")
	if tenantId == "" {
		err := errors.New("could not read tenant from environment")
		r.logger.Error(err, "could not read tenant from environment")
		return err
	}

	groupsClient, err := r.getGroupsClient()
	if err != nil {
		r.logger.Error(err, "Could not init groups client")
		return err
	}

	r.logger.Info("Creating AAD Group", "Group", cr.Spec.Routing.Group.Name)
	_, err = groupsClient.Create(context.TODO(), graphrbac.GroupCreateParameters{
		DisplayName:     to.StringPtr(cr.Spec.Routing.Group.Name),
		MailEnabled:     to.BoolPtr(false),
		MailNickname:    to.StringPtr(cr.Spec.Routing.Group.Name),
		SecurityEnabled: to.BoolPtr(true),
	})

	if err != nil {
		res, _ := err.(autorest.DetailedError)
		resStr, _ := ioutil.ReadAll(res.Response.Body)
		r.logger.Error(err, "Error on creating group", resStr)
	}

	return err
}

func (r *ReconcileRing) getGroupsClient() (*graphrbac.GroupsClient, error) {
	tenantId := os.Getenv("AZURE_TENANT_ID")
	if tenantId == "" {
		err := errors.New("could not read tenant from environment")
		return nil, err
	}

	authorizer, err := auth.NewAuthorizerFromEnvironmentWithResource("https://graph.windows.net")
	if err != nil {
		return nil, err
	}

	client := graphrbac.NewGroupsClientWithBaseURI("https://graph.windows.net", tenantId)
	client.Authorizer = authorizer
	return &client, nil
}

func (r *ReconcileRing) adGroupExists(cr *ringsv1alpha1.Ring) (bool, error) {
	// Check for production group
	// In this case, production is the set of "ALL" users, not a specific group
	// TODO - Add flag for making production an explicit group
	if cr.Spec.Routing.Group.Name == "*" {
		r.logger.Info("Listing requested for production group", "Group", cr.Spec.Routing.Group.Name)
		return true, nil
	}

	groupsClient, err := r.getGroupsClient()
	if err != nil {
		r.logger.Error(err, "Could not init groups client")
		return false, err
	}

	r.logger.Info("Listing AD Groups", "Group", cr.Spec.Routing.Group.Name)
	//res, err := groupsClient.List(context.TODO(), "")
	res, err := groupsClient.List(context.TODO(), fmt.Sprintf("mailNickname eq '%s'", cr.Spec.Routing.Group.Name))
	if err != nil {
		resStr, _ := ioutil.ReadAll(res.Response().Body)
		r.logger.Error(err, "Could not list AD Groups", "Group", cr.Spec.Routing.Group.Name, "Reason", resStr)
		return false, err
	}

	groups := res.Values()
	return len(groups) > 0, nil
}

func (r *ReconcileRing) deleteADGroup(cr *ringsv1alpha1.Ring) error {
	groupsClient, err := r.getGroupsClient()
	if err != nil {
		r.logger.Error(err, "Could not init groups client")
		return err
	}

	r.logger.Info("Deleting AAD Group", "Group", cr.Spec.Routing.Group)
	_, err = groupsClient.Delete(context.TODO(), "")

	return err
}
