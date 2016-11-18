# kube-aggregator
A server to unify Kubernetes API servers providing different resources for one cluster.

```bash
# start the kube apiserver
ALLOW_ANY_TOKEN=true ENABLE_RBAC=true ENABLE_AUTH_PROXY=true hack/local-up-cluster.sh

# change credentials for generated key to be used by the federator proxy
# TODO fix this to generate a key as someone other than root
sudo chmod 644 /var/run/kubernetes/client-auth-proxy.key

# start the federator
nice make && hack/local-up.sh



# create rbac roles and bindings for the api federator
echo `curl -k https://localhost:8444/bootstrap/rbac` | kubectl create -f - --validate=false --token=root/system:masters --server=https://localhost:6443

# bind the role you just created to the user `federation-editor` so that he can create api federation objects
kubectl create clusterrolebinding federator --clusterrole=apifederation.openshift.io:editor --user=federation-editor --token=root/system:masters  --server=https://localhost:6443

# create the api servers for the "normal" kube apiserver
kubectl create -f test/artifacts/default-kube-apiservers/ --token=federation-editor --server=https://localhost:8444

# grant yourself project-admin powers in every project, go through the federator to do it
# TODO this is broken until https://github.com/kubernetes/kubernetes/pull/36774 merges
kubectl create clusterrolebinding admin --clusterrole=admin --user=deads --token=root/system:masters  --server=https://localhost:8444

# log into the API federator as  yourself
# broken until https://github.com/openshift/origin/pull/11340 merges
oc login https://localhost:8444 --token deads

# this should be denied, because you don't have powers
kubectl get nodes --token=deads --server=https://localhost:8444

# this should succeed
kubectl get svc --token=deads --server=https://localhost:8444

```