package install

import (
	"github.com/openshift/kube-aggregator/pkg/apis/apifederation"
	"github.com/openshift/kube-aggregator/pkg/apis/apifederation/v1beta1"
	"k8s.io/kubernetes/pkg/apimachinery/announced"
	"k8s.io/kubernetes/pkg/util/sets"
)

func init() {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  apifederation.GroupName,
			RootScopedKinds:            sets.NewString("APIServer"),
			VersionPreferenceOrder:     []string{v1beta1.SchemeGroupVersion.Version},
			ImportPrefix:               "github.com/openshift/kube-aggregator/pkg/apis/apifederation",
			AddInternalObjectsToScheme: apifederation.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			v1beta1.SchemeGroupVersion.Version: v1beta1.AddToScheme,
		},
	).Announce().RegisterAndEnable(); err != nil {
		panic(err)
	}
}
