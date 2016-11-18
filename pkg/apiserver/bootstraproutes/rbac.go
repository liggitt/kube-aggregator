package bootstraproutes

import (
	"net/http"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/v1"
	rbac "k8s.io/kubernetes/pkg/apis/rbac"
	rbacv1alpha1 "k8s.io/kubernetes/pkg/apis/rbac/v1alpha1"
	"k8s.io/kubernetes/pkg/runtime"
)

// Index provides a webservice for the http root / listing all known paths.
type RoleBindings struct {
	AuthUser string
}

// Install adds the Index webservice to the given mux.
func (i RoleBindings) Install(mux *http.ServeMux) {
	mux.HandleFunc("/bootstrap/rbac", func(w http.ResponseWriter, r *http.Request) {
		resourceList := i.rbacResources()

		encoder := api.Codecs.LegacyCodec(rbacv1alpha1.SchemeGroupVersion, v1.SchemeGroupVersion)
		if err := runtime.EncodeList(encoder, resourceList.Items); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		objBytes, err := runtime.Encode(encoder, resourceList)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(objBytes)
	})
}

func (i RoleBindings) rbacResources() *api.List {
	ret := &api.List{}

	rolebindings := i.clusterRoleBindings()
	for i := range rolebindings {
		// TODO names need to be semi-unique.  Should probably update NewClusterRoleBinding to take a name
		rolebindings[i].Name = apifederationGroup + ":" + rolebindings[i].Name
		ret.Items = append(ret.Items, &rolebindings[i])
	}

	roles := i.clusterRoles()
	for i := range roles {
		ret.Items = append(ret.Items, &roles[i])
	}

	return ret
}

var (
	readWrite = []string{"get", "list", "watch", "create", "update", "patch", "delete", "deletecollection"}
	read      = []string{"get", "list", "watch"}
)

const (
	apifederationGroup = "apifederation.openshift.io"
)

func (i RoleBindings) clusterRoles() []rbac.ClusterRole {
	return []rbac.ClusterRole{
		{
			// a role which can manange apiservers
			ObjectMeta: api.ObjectMeta{Name: apifederationGroup + ":editor"},
			Rules: []rbac.PolicyRule{
				rbac.NewRule(readWrite...).Groups(apifederationGroup).Resources("apiservers").RuleOrDie(),
			},
		},
		{
			// a role which can read apiservers
			ObjectMeta: api.ObjectMeta{Name: apifederationGroup + ":reader"},
			Rules: []rbac.PolicyRule{
				rbac.NewRule(read...).Groups(apifederationGroup).Resources("apiservers").RuleOrDie(),
			},
		},
	}
}

func (i RoleBindings) clusterRoleBindings() []rbac.ClusterRoleBinding {
	return []rbac.ClusterRoleBinding{
		rbac.NewClusterBinding("system:auth-delegator").Users(i.AuthUser).BindingOrDie(),
	}
}
