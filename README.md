# A simple Kubernetes Operator to return VMware ESXi host information #

This repository contains a very simple Kubernetes Operator that uses VMware's govmomi to return some simple ESXi host information through the status field of a Custom Resource, HostInfo. This code is for education purposes only, showing one way in which the code in a Kubernetes controller can access the underlying vSphere resources for the purposes of querying those resources.
The CRD is built using [kubebuilder](https://go.kubebuilder.io/).

## What are we going to do in this tutorial? ##

In this example, we will create a CRD called HostInfo. HostInfo will contain the name of an ESXi host in its specification. When a Custom Resource object is created, and subsequently queried, the Total CPU and Free CPU from the ESXi host will be returned in the status fields of the yaml output.

The following will be created as part of this tutorial:

* A Customer Resource Definition (CRD)
  * Group: Topology
    * Kind: HostInfo
    * Version: V1
    * Spec.Hostname

* One of more HostInfo CR/Object manifests, each containing the name of an ESXi host that we wish to query.
  * Status.TotalCPU
  * Status.FreeCPU

## What is not covered in this tutorial? ##

The assumption is that you already have a working Kubernetes cluster. Installation and Deployment of Kubernetes is outside the scope of this tutorial. If you do not have a Kubernetes cluster consider using KinD, Kubernetes in Docker. A quickstaart guide can be found here:

* Kind (Kubernetes in Docker) - <https://kind.sigs.K8s.io/docs/user/quick-start/>

The assumption is that you also have a vSphere environment comprising of at least one ESXi host managed by a vCenter server. While the thought process is that your Kubernetes cluster will be running on vSphere infrastructure, and thus this operator will help you examine how the underlying vSphere resources are being consumed by the Kubernetes clusters running on top, it is not necessary for this to be the case for the purposes of this tutorial.

## What if I just want to understand some basic CRD concepts? ##

If this sounds even too daunting at this stage, I strongly recommend checking out the excellent tutorial on CRDs from my friend and colleague, Rafael Brito's [RockBand](https://github.com/brito-rafa/k8s-webhooks/blob/master/single-gvk/README.md) CRD demonstration. In that tutorial, he uses some very simple concepts to explains how CRDs work.

## Step 1 - Software Requirements ##

You will need the following components pre-installed on your desktop or workstation before we can build the CRD.

* git client - Apple Xcode or any git command line
* [Go (v1.13+)](https://golang.org/dl/)
* [Docker Desktop](https://www.docker.com/products/docker-desktop)
* [Kubebuilder](https://go.kubebuilder.io/quick-start.html)
* [Kustomize](https://kubernetes-sigs.github.io/kustomize/installation/)
* Access to a Container Image Repositor (docker.io, quay.io, harbor)

## Step 2 - KubeBuilder Scaffolding ##

I'm not going to spend a great deal of time talking about this. Suffice to say that KubeBuilder builds a template like experience for the creation of CRDs. We will build the scaffolding, and then add our own logic to query and return ESXi host CPU statistics as we go through the steps.

```cmd
mkdir hostinfo
$ cd hostinfo
```

Next, define the Go module name of your CRD. In my case, I have called it hostinfo. This create a go.mod file with the name of the module and the Go version (v1.15 here).

```cmd
$ go mod init hostinfo
go: creating new go.mod: module hostinfo
```

```cmd
$ ls
go.mod
```

```cmd
$ cat go.mod
module hostinfo

go 1.15
$
```

The following commands will create a directory structure which contains all the scaffolding necessary to build an operator. You may choose an alternate domain here.

```text
$ kubebuilder init --domain corinternal.com
Writing scaffold for you to edit...
Get controller runtime:
$ go get sigs.k8s.io/controller-runtime@v0.5.0
Update go.mod:
$ go mod tidy
Running make:
$ make
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...
go build -o bin/manager main.go
Next: define a resource with:
$ kubebuilder create api
$
```

As the output from the previous command states, we must now define a resource. To do that, we again use kubebuilder to create the resource, specifying the API group, its version and supported kind. My group is called topology, my kind is called HostInfo and my initial version is v1.

```text
$ kubebuilder create api --group topology --version v1 --kind HostInfo --resource=true --controller=true
Writing scaffold for you to edit...
api/v1/hostinfo_types.go
controllers/hostinfo_controller.go
Running make:
$ make
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...
go build -o bin/manager main.go
```

Our operator scaffolding (directory structure) is now in place. The next step is to implement our logic.

## Step 3 - Create the CRD ##

Customer Resource Definitions [CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) are a way to extend Kubernetes through Custom Resources. We are going to extend my Kubernetes cluster with a new custom resource called HostInfo which will retrieve information from an ESXi host. Thus, I will need to create a CRD which defines the specification of the custom resource, and the status fields that it returns. This is done by modifying the api/__version__/__crd__\_types.go file. In this tutorial, that file is called __api/v1/hostinfo_types.go__.

Here is the scaffolding provided by kubebuilder:

```go
// HostInfoSpec defines the desired state of HostInfo
type HostInfoSpec struct {
        // INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
        // Important: Run "make" to regenerate code after modifying this file

        // Foo is an example field of HostInfo. Edit HostInfo_types.go to remove/update
        Foo string `json:"foo,omitempty"`
}

// HostInfoStatus defines the observed state of HostInfo
type HostInfoStatus struct {
        // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
        // Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

```

Here is the file after it has been modified to include a single __spec.hostname__ field and to return two __status__ fields. There are also a number of kubebuilder fields added, which are used to do validation and other kubebuilder functions which are outside the scope of this discussion.

Note that what we are doing here is for education purposes only. Typically what you would observe is that the spec and status fields would be similar, and it is the function of the controller to reconcile and differences between the two to achieve eventual consistency.

```go
// HostInfoSpec defines the desired state of HostInfo
type HostInfoSpec struct {
        Hostname string `json:"hostname"`
}

// HostInfoStatus defines the observed state of HostInfo
type HostInfoStatus struct {
        TotalCPU int64 `json:"totalCPU"`
        FreeCPU  int64 `json:"freeCPU"`
}

// +kubebuilder:validation:Optional
// +kubebuilder:resource:shortName={"ch"}
// +kubebuilder:printcolumn:name="Hostname",type=string,JSONPath=`.spec.hostname`
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

```

We are now ready to create the CRD. There is one final step however, and this involves updating the Makefile. In the default Makefile, the following CRD appears:

```Makefile
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
```

This CRD_OPTIONS entry will need to be changed to the following:

```Makefile
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:preserveUnknownFields=false,crdVersions=v1,trivialVersions=true"
```

Now we can build our CRD based on the spec and status fields that we have place in the __api/v1/hostinfo_types.go__ file.

```text
$ make manifests && make generate
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
```

## Step 4 - Install the CRD ##

The CRD is not currently installed. Let's do that next.

```text
$ kubectl get crd
NAME                                                               CREATED AT
antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com        2020-11-18T17:14:03Z
antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com   2020-11-18T17:14:03Z
clusternetworkpolicies.security.antrea.tanzu.vmware.com            2020-11-18T17:14:03Z
traceflows.ops.antrea.tanzu.vmware.com                             2020-11-18T17:14:03Z
```

```text
$ make install
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
kustomize build config/crd | kubectl apply -f -
customresourcedefinition.apiextensions.k8s.io/hostinfoes.topology.corinternal.com created
```

```text
$ kubectl get crd
NAME                                                               CREATED AT
antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com        2020-11-18T17:14:03Z
antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com   2020-11-18T17:14:03Z
clusternetworkpolicies.security.antrea.tanzu.vmware.com            2020-11-18T17:14:03Z
hostinfoes.topology.corinternal.com                                2020-12-31T15:30:17Z
traceflows.ops.antrea.tanzu.vmware.com                             2020-11-18T17:14:03Z
```

Another useful way to check is to use the following command against our API group. Remember back in step 2 we specified the domain as __corinternal.com__ and the group as __topology__.

```text
$ kubectl api-resources --api-group=topology.corinternal.com
NAME         SHORTNAMES   APIGROUP                   NAMESPACED   KIND
hostinfoes   ch           topology.corinternal.com   true         HostInfo
```

## Step 5 - Test the CRD ##

At this point, we can do a quick test to see if our CRD is in fact working. To do that, we can create a manifest file and see if we can instantiate such an object (or custom resource) on our Kubernetes cluster. Fortunately kubebuilder provides us with a sample manifest which can be found in __config/samples__.

```text
$ cd config/samples
$ ls
topology_v1_hostinfo.yaml
```

```yaml
$ cat topology_v1_hostinfo.yaml
apiVersion: topology.corinternal.com/v1
kind: HostInfo
metadata:
  name: hostinfo-sample
spec:
  # Add fields here
  foo: bar
```

We need to modify this sample manifest so that the specification matches what we added to our CRD. We had a __spec.hostname__ field, as per the __api/v1/hostinfo_types.go__ modification. Thus, after a simple modification, the CR manifest looks like this, where esxi-dell-e.rainpole.com is the name of the ESXi host that I wish to query.

```yaml
$ cat topology_v1_hostinfo.yaml
apiVersion: topology.corinternal.com/v1
kind: HostInfo
metadata:
  name: hostinfo-host-e
spec:
  # Add fields here
  hostname: esxi-dell-e.rainpole.com
```

To see if it works, we need to create this Custom Resource.

```text
$ kubectl create -f topology_v1_hostinfo.yaml
hostinfo.topology.corinternal.com/hostinfo-host-e created
```

```text
$ kubectl get hostinfo
NAME              HOSTNAME
hostinfo-host-e   esxi-dell-e.rainpole.com
```

```yaml
$ kubectl get hostinfo -o yaml
apiVersion: v1
items:
- apiVersion: topology.corinternal.com/v1
  kind: HostInfo
  metadata:
    creationTimestamp: "2020-12-31T15:48:49Z"
    generation: 1
    managedFields:
    - apiVersion: topology.corinternal.com/v1
      fieldsType: FieldsV1
      fieldsV1:
        f:spec:
          .: {}
          f:hostname: {}
      manager: kubectl
      operation: Update
      time: "2020-12-31T15:48:49Z"
    name: hostinfo-host-e
    namespace: default
    resourceVersion: "20716173"
    selfLink: /apis/topology.corinternal.com/v1/namespaces/default/hostinfoes/hostinfo-host-e
    uid: c7ff0546-b9f0-49b5-8ea6-748b1f10d039
  spec:
    hostname: esxi-dell-e.rainpole.com
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

## Step 6 - Create the controller / manager ##

This appears to be working as expected. However note that there are no Status fields with our CPU information in the output. We need to implement our controller to do this. The controller implements your desired business login. In the logic of my controller, I first read the vCenter server credentials and access details from a Kubernetes secret (which we will create shortly). I will then open a session to my vCenter server, and get a list of ESXi hosts that it manages. I then look for the ESXi host that is specified in the CR, and retrieve the Total CPU and Free CPU statistics for this host, and update the appropriate Status field with this information.

Once all this business logic has been added, I will build a container image which contains my controller. This will then be provisioned as a deployment containing a single Pod with two containers (more on this shortly). The deployment ensures that my Pod is restarted in the event of a failure.

This is what kubebuilder provides as controller scaffolding - we are most interestedin the __HostInfoReconciler__ function:

```go
func (r *HostInfoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
        _ = context.Background()
        _ = r.Log.WithValues("hostinfo", req.NamespacedName)

        // your logic here

        return ctrl.Result{}, nil
}
```

Considering the business logic that I described above, this is what my updated __HostInfoReconciler__ function looks like. Hopefully the comments make is easy to understand, but at the end of the day, when this controller gets a Reconcile request, the TotalCPU and FreeCPU fields in the status of the Custom Resource are update for the specific ESXi host. Note that I have omitted a number of required imports that also need to be added to the controller. Refer to the code for the complete __hostinfo_controller.go__ code.

```go
func (r *HostInfoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

        ctx := context.Background()
        log := r.Log.WithValues("hostinfo", req.NamespacedName)

        ch := &topologyv1.HostInfo{}
        if err := r.Client.Get(ctx, req.NamespacedName, ch); err != nil {
                // add some debug information if it's not a NotFound error
                if !k8serr.IsNotFound(err) {
                        log.Error(err, "unable to fetch HostInfo")
                }
                return ctrl.Result{}, client.IgnoreNotFound(err)
        }

        msg := fmt.Sprintf("received reconcile request for %q (namespace: %q)", ch.GetName(), ch.GetNamespace())
        log.Info(msg)

        // We will retrieve these environment variables through 
        // passing 'secret' parameters via the manager manifest

        vc := os.Getenv("GOVMOMI_URL")
        user := os.Getenv("GOVMOMI_USERNAME")
        pwd := os.Getenv("GOVMOMI_PASSWORD")

        //
        // Create a vSphere/vCenter client
        //
        //    The govmomi client requires a URL object, u, 
        //    not just a string representation of the vCenter URL
        //

        u, err := soap.ParseURL(vc)

        if err != nil {
                msg := fmt.Sprintf("unable to parse vCenter URL: error %s", err)
                log.Info(msg)
                return ctrl.Result{}, err
        }

        u.User = url.UserPassword(user, pwd)

        // Share govc's session cache (from https://github.com/vmware/govmomi/blob/master/examples/examples.go)
        s := &cache.Session{
                URL:      u,
                Insecure: true,
        }

        c := new(vim25.Client)

        err = s.Login(ctx, c, nil)

        if err != nil {
                msg := fmt.Sprintf("unable to login to vCenter: error %s", err)
                log.Info(msg)
                return ctrl.Result{}, err
        }

        //
        // Create a view manager
        //

        m := view.NewManager(c)

        //
        // Create a container view of HostSystem objects
        //

        v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"HostSystem"}, true)

        if err != nil {
                msg := fmt.Sprintf("unable to create container view for HostSystem: error %s", err)
                log.Info(msg)
                return ctrl.Result{}, err
        }

        defer v.Destroy(ctx)

        //
        // Retrieve summary property for all hosts
        //

        var hss []mo.HostSystem

        err = v.Retrieve(ctx, []string{"HostSystem"}, []string{"summary"}, &hss)

        if err != nil {
                msg := fmt.Sprintf("unable to retrieve HostSystem summary: error %s", err)
                log.Info(msg)
                return ctrl.Result{}, err
        }

        //
        // Print summary for host in HostInfo specification info
        //

        for _, hs := range hss {
                if hs.Summary.Config.Name == ch.Spec.Hostname {
                        ch.Status.TotalCPU = int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
                        ch.Status.FreeCPU = (int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)) - int64(hs.Summary.QuickStats.OverallCpuUsage)
                }
        }

        if err := r.Status().Update(ctx, ch); err != nil {
                log.Error(err, "unable to update HostInfo status")
                return ctrl.Result{}, err
        }

        return ctrl.Result{}, nil
}
```

We can now proceed to build the controller / manager.

## Step 7 - Build the controller ##

At this point everything is in place to enable us to deploy the controller to the Kubernete cluster. If you remember back to the prerequisites in step 1, we said that you need access to a container image registry. This is where this comes in. The Dockerfile with the appropriate directives is already in place. MAke sure you login to your image repository, i.e. docker login, before proceeding.

To create the container image of our controller / manager, simply run the following commands:

```text
$ make docker-build docker-push IMG=docker.io/cormachogan/hostinfo-controller:v1
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...
/usr/share/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
go test ./... -coverprofile cover.out
?       hostinfo        [no test files]
?       hostinfo/api/v1 [no test files]
ok      hostinfo/controllers    8.401s  coverage: 0.0% of statements
docker build . -t docker.io/cormachogan/hostinfo-controller:v1
Sending build context to Docker daemon  53.31MB
Step 1/14 : FROM golang:1.13 as builder
 ---> d6f3656320fe
Step 2/14 : WORKDIR /workspace
 ---> Running in 30a535f6a3de
Removing intermediate container 30a535f6a3de
 ---> 0f6c055c6fc8
Step 3/14 : COPY go.mod go.mod
 ---> 11d0f2eda936
Step 4/14 : COPY go.sum go.sum
 ---> ccec3c47ed5a
Step 5/14 : RUN go mod download
 ---> Running in a25193d9d72c
go: finding cloud.google.com/go v0.38.0
go: finding github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78
go: finding github.com/Azure/go-autorest/autorest v0.9.0
go: finding github.com/Azure/go-autorest/autorest/adal v0.5.0
.
.
go: finding sigs.k8s.io/controller-runtime v0.5.0
go: finding sigs.k8s.io/structured-merge-diff v1.0.1-0.20191108220359-b1b620dd3f06
go: finding sigs.k8s.io/yaml v1.1.0
Removing intermediate container a25193d9d72c
 ---> 7e556d5ee595
Step 6/14 : COPY main.go main.go
 ---> 1f0a5564360d
Step 7/14 : COPY api/ api/
 ---> 658146b97c2e
Step 8/14 : COPY controllers/ controllers/
 ---> 5c494bc11a2d
Step 9/14 : RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go
 ---> Running in 39a4ae69c02d
Removing intermediate container 39a4ae69c02d
 ---> 465a22e4df85
Step 10/14 : FROM gcr.io/distroless/static:nonroot
 ---> aa99000bc55d
Step 11/14 : WORKDIR /
 ---> Using cache
 ---> 8bcbc4c15403
Step 12/14 : COPY --from=builder /workspace/manager .
 ---> 9323cb1f88c5
Step 13/14 : USER nonroot:nonroot
 ---> Running in 0d85b3457944
Removing intermediate container 0d85b3457944
 ---> 7d038e0d82f5
Step 14/14 : ENTRYPOINT ["/manager"]
 ---> Running in 5f5569796b9a
Removing intermediate container 5f5569796b9a
 ---> 05133c0de2d9
Successfully built 05133c0de2d9
Successfully tagged cormachogan/hostinfo-controller:v1
docker push docker.io/cormachogan/hostinfo-controller:v1
The push refers to repository [docker.io/cormachogan/hostinfo-controller]
5758f4a008b9: Pushed
7a5b9c0b4b14: Pushed
v1: digest: sha256:f970a9610304c885ffd03edc0c7ddd485fb399279511054a578ade406224ad6b size: 739
$
```

The container image of the controller is now built and pushed.

## Step 8 - Modify the Manager manifest to include environment variables ##

Kubebuilder provides a manager manifest scaffold file. However, since we need to provide vCenter details to our controller, we need to add these to our manager manifest file, which is found in __config/manager/manager.yaml__. This is the deployment manifest for our controller. In the spec, we need to add an additional __spec.env__ section which has the environment variables defined, as well as the name of our secret (which we will create shortly).

```yaml
    spec:
      .
      .
        env:
          - name: GOVMOMI_USERNAME
            valueFrom:
              secretKeyRef:
                name: vc-creds
                key: GOVMOMI_USERNAME
          - name: GOVMOMI_PASSWORD
            valueFrom:
              secretKeyRef:
                name: vc-creds
                key: GOVMOMI_PASSWORD
          - name: GOVMOMI_URL
            valueFrom:
              secretKeyRef:
                name: vc-creds
                key: GOVMOMI_URL
      volumes:
        - name: vc-creds
          secret:
            secretName: vc-creds
      terminationGracePeriodSeconds: 10
```

Note that the secret needs to be deployed in the same namespace that the controller is going to run in,
__hostinfo-system__. Thus, the secret is created using the following command:

```text
$ kubectl create ns hostinfo-system
namespace/hostinfo-system created
```

```text
$ kubectl create secret generic vc-creds \
--from-literal='GOVMOMI_USERNAME=administrator@vsphere.local' \
--from-literal='GOVMOMI_PASSWORD=VMware123!' \
--from-literal='GOVMOMI_URL=192.168.0.100' \
-n hostinfo-system
namespace/hostinfo-system created
```

We are now ready to deploy the controller to the Kubernetes cluster.

## Step 9 - Deploy the controller ##

```text
$ make deploy IMG=docker.io/cormachogan/hostinfo-controller:v1
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cd config/manager && kustomize edit set image controller=docker.io/cormachogan/hostinfo-controller:v1
kustomize build config/default | kubectl apply -f -
namespace/hostinfo-system unchanged
customresourcedefinition.apiextensions.k8s.io/hostinfoes.topology.corinternal.com configured
role.rbac.authorization.k8s.io/hostinfo-leader-election-role unchanged
clusterrole.rbac.authorization.k8s.io/hostinfo-manager-role configured
clusterrole.rbac.authorization.k8s.io/hostinfo-proxy-role unchanged
clusterrole.rbac.authorization.k8s.io/hostinfo-metrics-reader unchanged
rolebinding.rbac.authorization.k8s.io/hostinfo-leader-election-rolebinding unchanged
clusterrolebinding.rbac.authorization.k8s.io/hostinfo-manager-rolebinding unchanged
clusterrolebinding.rbac.authorization.k8s.io/hostinfo-proxy-rolebinding unchanged
service/hostinfo-controller-manager-metrics-service unchanged
deployment.apps/hostinfo-controller-manager configured
$
```

## Step 10 - Check controller functionality ##

Now that our controller has been deployed, let's see if it is working.

### Step 10.1 - Check the deployment ###

```text
$ kubectl get deploy -n hostinfo-system
NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
hostinfo-controller-manager   1/1     1            1           14m
```

### Step 10.2 - Check the Pod ###

```text
$ kubectl get pods -n hostinfo-system
NAME                                           READY   STATUS    RESTARTS   AGE
hostinfo-controller-manager-6484c486ff-8vwsn   2/2     Running   0          72s
```

### Step 10.3 - Check the controller / manager logs ###

```text
$ kubectl logs hostinfo-controller-manager-6484c486ff-8vwsn -n hostinfo-system manager
2020-12-31T16:54:55.633Z        INFO    controller-runtime.metrics      metrics server is starting to listen    {"addr": "127.0.0.1:8080"}
2020-12-31T16:54:55.634Z        INFO    setup   starting manager
I1231 16:54:55.634543       1 leaderelection.go:242] attempting to acquire leader lease  hostinfo-system/0df5945b.corinternal.com...
2020-12-31T16:54:55.635Z        INFO    controller-runtime.manager      starting metrics server {"path": "/metrics"}
I1231 16:55:13.035397       1 leaderelection.go:252] successfully acquired lease hostinfo-system/0df5945b.corinternal.com
2020-12-31T16:55:13.035Z        DEBUG   controller-runtime.manager.events       Normal  {"object": {"kind":"ConfigMap","namespace":"hostinfo-system","name":"0df5945b.corinternal.com","uid":"f1f46185-77f5-43d2-ba33-192caed82409","apiVersion":"v1","resourceVersion":"20735459"}, "reason": "LeaderElection", "message": "hostinfo-controller-manager-6484c486ff-8vwsn_510f151d-4e35-4f42-966e-31ddcec34bcb became leader"}
2020-12-31T16:55:13.035Z        INFO    controller-runtime.controller   Starting EventSource    {"controller": "hostinfo", "source": "kind source: /, Kind="}
2020-12-31T16:55:13.135Z        INFO    controller-runtime.controller   Starting Controller     {"controller": "hostinfo"}
2020-12-31T16:55:13.135Z        INFO    controller-runtime.controller   Starting workers        {"controller": "hostinfo", "worker count": 1}
2020-12-31T16:55:13.136Z        INFO    controllers.HostInfo    received reconcile request for "hostinfo-host-e" (namespace: "default") {"hostinfo": "default/hostinfo-host-e"}
2020-12-31T16:55:13.625Z        DEBUG   controller-runtime.controller   Successfully Reconciled {"controller": "hostinfo", "request": "default/hostinfo-host-e"}
2020-12-31T16:55:13.625Z        INFO    controllers.HostInfo    received reconcile request for "hostinfo-host-e" (namespace: "default") {"hostinfo": "default/hostinfo-host-e"}
```

### Step 10.4 - Check if CPU statistics are returned in the status ###

```text
$ kubectl get hostinfo hostinfo-host-e -o yaml
apiVersion: topology.corinternal.com/v1
kind: HostInfo
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"topology.corinternal.com/v1","kind":"HostInfo","metadata":{"annotations":{},"name":"hostinfo-host-e","namespace":"default"},"spec":{"hostname":"esxi-dell-e.rainpole.com"}}
  creationTimestamp: "2020-12-31T15:48:49Z"
  generation: 1
  managedFields:
  - apiVersion: topology.corinternal.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
      f:spec:
        .: {}
        f:hostname: {}
    manager: kubectl
    operation: Update
    time: "2020-12-31T16:46:51Z"
  - apiVersion: topology.corinternal.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:status:
        .: {}
        f:freeCPU: {}
        f:totalCPU: {}
    manager: manager
    operation: Update
    time: "2020-12-31T16:55:13Z"
  name: hostinfo-host-e
  namespace: default
  resourceVersion: "20735464"
  selfLink: /apis/topology.corinternal.com/v1/namespaces/default/hostinfoes/hostinfo-host-e
  uid: c7ff0546-b9f0-49b5-8ea6-748b1f10d039
spec:
  hostname: esxi-dell-e.rainpole.com
status:
  freeCPU: 40514
  totalCPU: 43980
```

Success!!! Note that the output above is showing us freeCPU and totalCPU as per our business logic implemented in the controller. How cool is that? You can now go ahead and create additional HostInfo manifests for different hosts in your vSphere environment managed by your vCenter server, and you should be able to get free and total CPU from those as well.

## What next? ##

If this has given you a desire to do more exciting stuff with Kubernetes Operators on top of vSphere, check out the [vmgroup](https://github.com/embano1/codeconnect-vm-operator/blob/main/README.md) operator that my other colleague Micheal Gasch created. It will let you deploy virtual machines on vSphere via a Kubernetes operator. Cool stuff for sure.
