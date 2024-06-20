package logserver

import (
	"context"
	"fmt"

	"github.com/securesign/operator/internal/controller/common/action"
	k8sutils "github.com/securesign/operator/internal/controller/common/utils/kubernetes"
	"github.com/securesign/operator/internal/controller/constants"
	"github.com/securesign/operator/internal/controller/trillian/actions"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	rhtasv1alpha1 "github.com/securesign/operator/api/v1alpha1"
)

func NewCreateRouteAction() action.Action[rhtasv1alpha1.Trillian] {
	return &createRouteAction{}
}

type createRouteAction struct {
	action.BaseAction
}

func (i createRouteAction) Name() string {
	return "create route"
}

func (i createRouteAction) CanHandle(ctx context.Context, instance *rhtasv1alpha1.Trillian) bool {
	c := meta.FindStatusCondition(instance.Status.Conditions, constants.Ready)
	return c.Reason == constants.Creating || c.Reason == constants.Ready // TODO: check if ctlog is external
}

func (i createRouteAction) Handle(ctx context.Context, instance *rhtasv1alpha1.Trillian) *action.Result {

	var (
		err     error
		updated bool
	)

	labels := constants.LabelsFor(actions.LogServerComponentName, actions.LogserverDeploymentName, instance.Name)
	logserverRoute := k8sutils.CreateRoute(instance.Namespace, actions.LogserverDeploymentName, labels)

	if err = controllerutil.SetControllerReference(instance, logserverRoute, i.Client.Scheme()); err != nil {
		return i.Failed(fmt.Errorf("could not set controller reference for logserver Route: %w", err))
	}

	if updated, err = i.Ensure(ctx, logserverRoute); err != nil {
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    actions.ServerCondition,
			Status:  metav1.ConditionFalse,
			Reason:  constants.Failure,
			Message: err.Error(),
		})
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    constants.Ready,
			Status:  metav1.ConditionFalse,
			Reason:  constants.Failure,
			Message: err.Error(),
		})
		return i.FailedWithStatusUpdate(ctx, fmt.Errorf("could not create logserver Route: %w", err), instance)
	}

	if updated {
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    actions.ServerCondition,
			Status:  metav1.ConditionFalse,
			Reason:  constants.Creating,
			Message: "Route created",
		})
		return i.StatusUpdate(ctx, instance)
	} else {
		return i.Continue()
	}

}
