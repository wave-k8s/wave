package deployment

import (
	"context"

	"github.com/wave-k8s/wave/pkg/core"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-apps-v1-deployment,mutating=true,failurePolicy=ignore,groups=apps,resources=deployments,verbs=create;update,versions=v1,name=deployments.wave.pusher.com,admissionReviewVersions=v1,sideEffects=NoneOnDryRun

type DeploymentWebhook struct {
	client.Client
	Handler *core.Handler
}

func (a *DeploymentWebhook) Default(ctx context.Context, obj runtime.Object) error {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return err
	}
	err = a.Handler.HandleDeploymentWebhook(obj.(*appsv1.Deployment), request.DryRun, request.Operation == "CREATE")
	return err
}

func AddDeploymentWebhook(mgr manager.Manager) error {
	err := builder.WebhookManagedBy(mgr).For(&appsv1.Deployment{}).WithDefaulter(
		&DeploymentWebhook{
			Client:  mgr.GetClient(),
			Handler: core.NewHandler(mgr.GetClient(), mgr.GetEventRecorderFor("wave")),
		}).Complete()

	return err
}
