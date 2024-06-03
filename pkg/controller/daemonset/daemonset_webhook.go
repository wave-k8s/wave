package daemonset

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

// +kubebuilder:webhook:path=/mutate-apps-v1-daemonset,mutating=true,failurePolicy=ignore,groups=apps,resources=daemonsets,verbs=create;update,versions=v1,name=daemonsets.wave.pusher.com,admissionReviewVersions=v1,sideEffects=NoneOnDryRun

type DaemonSetWebhook struct {
	client.Client
	Handler *core.Handler[*appsv1.DaemonSet]
}

func (a *DaemonSetWebhook) Default(ctx context.Context, obj runtime.Object) error {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return err
	}
	err = a.Handler.HandleWebhook(obj.(*appsv1.DaemonSet), request.DryRun, request.Operation == "CREATE")
	return err
}

func AddDaemonSetWebhook(mgr manager.Manager) error {
	err := builder.WebhookManagedBy(mgr).For(&appsv1.DaemonSet{}).WithDefaulter(
		&DaemonSetWebhook{
			Client:  mgr.GetClient(),
			Handler: core.NewHandler[*appsv1.DaemonSet](mgr.GetClient(), mgr.GetEventRecorderFor("wave")),
		}).Complete()

	return err
}
