# A simple Kubernetes Operator to return VMware ESXi host information #

This repository contains a very simple Kubernetes Operator that uses VMware's __govmomi__ to return some simple ESXi host information through the status field of a __Custom Resource (CR)__, which is called ```HostInfo```. This will require us to extend Kubernetes with a new __Custom Resource Definition (CRD)__. The code shown here is for education purposes only, showing one way in which a Kubernetes controller / operator can access the underlying vSphere infrastructure for the purposes of querying resources.

You can think of a CRD as representing the desired state of a Kubernetes object or Custom Resource, and the function of the operator is to run the logic or code to make that desired state happen - in other words the operator has the logic to do whatever is necessary to achieve the object's desired state.

## What are we going to do in this tutorial? ##

In this example, we will create a CRD called ```HostInfo```. HostInfo will contain the name of an ESXi host in its specification. When a Custom Resource (CR) is created and subsequently queried, we will call an operator (logic in a controller) whereby the Total CPU and Free CPU from the ESXi host will be returned via the status fields of the object through govmomi API calls.

The following will be created as part of this tutorial:

* A __Customer Resource Definition (CRD)__
  * Group: ```Topology```
    * Kind: ```HostInfo```
    * Version: ```v1```
    * Specification will include a single item: ```Spec.Hostname```

* One or more __HostInfo Custom Resource / Object__ will be created through yaml manifests, each manifest containing the hostname of an ESXi host that we wish to query. The fields which will be updated to contain the relevant information from the ESXi host (when the CR is queried) are:
  * ```Status.TotalCPU```
  * ```Status.FreeCPU```

* An __Operator__ (or business logic) to retrieve the Total and Free CPU from the ESXi host specified in the CR will be coded in the controller for this CR.

__Note:__ A similar exercise to create an operator to query virtual machine information. This can be found [here](https://github.com/cormachogan/vminfo-operator).

## What is not covered in this tutorial? ##

The assumption is that you already have a working Kubernetes cluster. Installation and deployment of a Kubernetes is outside the scope of this tutorial. If you do not have a Kubernetes cluster available, consider using __Kubernetes in Docker__ (shortened to __Kind__) which uses containers as Kubernetes nodes. A quickstart guide can be found here:

* [Kind (Kubernetes in Docker)](https://kind.sigs.K8s.io/docs/user/quick-start/)

The assumption is that you also have a __VMware vSphere environment__ comprising of at least one ESXi hypervisor which is managed by a vCenter server. While the thought process is that your Kubernetes cluster will be running on vSphere infrastructure, and thus this operator will help you examine how the underlying vSphere resources are being consumed by the Kubernetes clusters running on top, it is not necessary for this to be the case for the purposes of this tutorial. You can use this code to query any vSphere environment from Kubernetes.

## What if I just want to understand some basic CRD concepts? ##

If this sounds even too daunting at this stage, I strongly recommend checking out the excellent tutorial on CRDs from my colleague, __Rafael Brito__. His [RockBand](https://github.com/brito-rafa/k8s-webhooks/blob/master/single-gvk/README.md) CRD tutorial uses some very simple concepts to explain how CRDs, CRs, Operators, spec and status fields work.

## Step 1 - Software Requirements ##

You will need the following components pre-installed on your desktop or workstation before we can build the CRD and operator.

* A __git__ client/command line
* [Go (v1.15+)](https://golang.org/dl/) - earlier versions may work but I used v1.15.
* [Docker Desktop](https://www.docker.com/products/docker-desktop)
* [Kubebuilder](https://go.kubebuilder.io/quick-start.html)
* [Kustomize](https://kubernetes-sigs.github.io/kustomize/installation/)
* Access to a Container Image Repositor (docker.io, quay.io, harbor)
* A __make__ binary - used by Kubebuilder

If you are interested in learning more about Golang basics, I found [this site](https://tour.golang.org/welcome/1) very helpful.

## Step 2 - KubeBuilder Scaffolding ##

The CRD is built using [kubebuilder](https://go.kubebuilder.io/).  I'm not going to spend a great deal of time talking about __KubeBuilder__. Suffice to say that KubeBuilder builds a directory structure containing all of the templates (or scaffolding) necessary for the creation of CRDs. Once this scaffolding is in place, this turorial will show you how to add your own specification fields and status fields, as well as how to add your own operator logic. In this example, our logic will login to vSphere, query and return ESXi host CPU statistics via a Kubernetes CR / object / Kind called HostInfo, the values of which will be used to populate status fields in our CRs.

The following steps will create the scaffolding to get started.

```cmd
mkdir hostinfo
$ cd hostinfo
```

Next, define the Go module name of your CRD. In my case, I have called it __hostinfo__. This creates a __go.mod__ file with the name of the module and the Go version (v1.15 here).

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
```

Now we can proceed with building out the rest of the directory structure. The following __kubebuilder__ commands (__init__ and __create api__) creates all the scaffolding necessary to build our CRD and operator. You may choose an alternate __domain__ here if you wish. Simply make note of it as you will be referring to it later in the tutorial.

```cmd
kubebuilder init --domain corinternal.com
```

Here is what the output from the command looks like:

```cmd
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

```cmd
kubebuilder create api \
--group topology       \
--version v1           \
--kind HostInfo        \
--resource=true        \
--controller=true
```

Here is the output from that command:

```cmd
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

Our operator scaffolding (directory structure) is now in place. The next step is to define the specification and status fields in our CRD. After that, we create the controller logic which will watch our Custom Resources, and bring them to desired state (called a reconcile operation). More on this shortly.

## Step 3 - Create the CRD ##

Customer Resource Definitions [CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) are a way to extend Kubernetes through Custom Resources. We are going to extend a Kubernetes cluster with a new custom resource called __HostInfo__ which will retrieve information from an ESXi host placed whose name is specified in a Custom Resource. Thus, I will need to create a field called hostname in the CRD - this defines the specification of the custom resource. We also add two status fields, as these will be used to return information like TotalCPU and FreeCPU from the ESXi host.

This is done by modifying the __api/v1/hostinfo_types.go__ file. Here is the initial scaffolding / template provided by kubebuilder:

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

This file is modified to include a single __spec.hostname__ field and to return two __status__ fields. There are also a number of kubebuilder fields added, which are used to do validation and other kubebuilder related functions. The shortname "ch" will be used later on in our controller logic. Also, when we query any Custom Resources created with the CRD, e.g. ```kubectl get hostinfo```, we want the output to display the hostname of the ESXi host.

Note that what we are doing here is for education purposes only. Typically what you would observe is that the spec and status fields would be similar, and it is the function of the controller to reconcile and differences between the two to achieve eventual consistency. But we are keeping things simple, as the purpose here is to show how vSphere can be queried from a Kubernetes Operator. Below is a snippet of the __hostinfo_types.go__ showing the code changes. The code-complete [hostinfo_types.go](api/v1/hostinfo_types.go) is here.

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

We are now ready to create the CRD. There is one final step however, and this involves updating the __Makefile__ which kubebuilder has created for us. In the default Makefile created by kubebuilder, the following __CRD_OPTIONS__ line appears:

```Makefile
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
```

This CRD_OPTIONS entry should be changed to the following:

```Makefile
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:preserveUnknownFields=false,crdVersions=v1,trivialVersions=true"
```

Now we can build our CRD with the spec and status fields that we have place in the __api/v1/hostinfo_types.go__ file.

```cmd
make manifests && make generate
```

Here is the output from the make:

```Makefile
$ make manifests && make generate
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
```

## Step 4 - Install the CRD ##

The CRD is not currently installed in the Kubernetes Cluster.

```shell
$ kubectl get crd
NAME                                                               CREATED AT
antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com        2020-11-18T17:14:03Z
antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com   2020-11-18T17:14:03Z
clusternetworkpolicies.security.antrea.tanzu.vmware.com            2020-11-18T17:14:03Z
traceflows.ops.antrea.tanzu.vmware.com                             2020-11-18T17:14:03Z
```

To install the CRD, run the following make command:

```cmd
make install
```

The output should look something like this:

```makefile
$ make install
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
kustomize build config/crd | kubectl apply -f -
customresourcedefinition.apiextensions.k8s.io/hostinfoes.topology.corinternal.com created
```

Now check to see if the CRD is installed running the same command as before.

```shell
$ kubectl get crd
NAME                                                               CREATED AT
antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com        2020-11-18T17:14:03Z
antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com   2020-11-18T17:14:03Z
clusternetworkpolicies.security.antrea.tanzu.vmware.com            2020-11-18T17:14:03Z
hostinfoes.topology.corinternal.com                                2020-12-31T15:30:17Z
traceflows.ops.antrea.tanzu.vmware.com                             2020-11-18T17:14:03Z
```

Our new CRD ```hostinfoes.topology.corinternal.com``` is now visible. Another useful way to check if the CRD has successfully deployed is to use the following command against our API group. Remember back in step 2 we specified the domain as ```corinternal.com``` and the group as ```topology```. Thus the command to query api-resources for this CRD is as follows:

```shell
$ kubectl api-resources --api-group=topology.corinternal.com
NAME         SHORTNAMES   APIGROUP                   NAMESPACED   KIND
hostinfoes   ch           topology.corinternal.com   true         HostInfo
```

## Step 5 - Test the CRD ##

At this point, we can do a quick test to see if our CRD is in fact working. To do that, we can create a manifest file with a Custom Resource that uses our CRD, and see if we can instantiate such an object (or custom resource) on our Kubernetes cluster. Fortunately kubebuilder provides us with a sample manifest that we can use for this. It can be found in __config/samples__.

```shell
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

We need to slightly modify this sample manifest so that the specification field matches what we added to our CRD. Note the spec: above where it states 'Add fields here'. We have removed the __foo__ field and added a __spec.hostname__ field, as per the __api/v1/hostinfo_types.go__ modification earlier. Thus, after a simple modification, the CR manifest looks like this, where __esxi-dell-e.rainpole.com__ is the name of the ESXi host that we wish to query.

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

To see if it works, we need to create this HostInfo Custom Resource.

```shell
$ kubectl create -f topology_v1_hostinfo.yaml
hostinfo.topology.corinternal.com/hostinfo-host-e created
```

```shell
$ kubectl get hostinfo
NAME              HOSTNAME
hostinfo-host-e   esxi-dell-e.rainpole.com
```

Note that the hostname field is also printed, as per the kubebuilder directive that we placed in the __api/v1/hostinfo_types.go__. As a final test, we will display the CR in yaml format.

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

This appears to be working as expected. However there are no __Status__ fields displayed with our CPU information in the __yaml__ output above. To see this information, we need to implement our operator / controller logic to do this. The controller implements the desired business logic. In this controller, we first read the vCenter server credentials from a Kubernetes secret (which we will create shortly). We will then open a session to my vCenter server, and get a list of ESXi hosts that it manages. I then look for the ESXi host that is specified in the spec.hostname field in the CR, and retrieve the Total CPU and Free CPU statistics for this host. Finally we will update the appropriate Status field with this information, and we should be able to query it using the __kubectl get hostinfo -o yaml__ command seen previously.

__Note:__ As has been pointed out, this code is not very optomized, and logging into vCenter Server for every reconcile request is not ideal. The login function should be moved out of the reconcile request, and it is something I will look at going forward. But for our present learning purposes, its fine to do this as we won't be overloading the vCenter Server with our handful of reconcile requests.

Once all this business logic has been added in the controller, we will need to be able to run it in the Kubernetes cluster. To achieve this, we will build a container image to run the controller logic. This will be provisioned in the Kubernetes cluster using a Deployment manifest. The deployment contains a single Pod that runs the container (it is called __manager__). The deployment ensures that my Pod is restarted in the event of a failure.

This is what kubebuilder provides as controller scaffolding - it is found in __controllers/hostinfo_controller.go__ - we are most interested in the __HostInfoReconciler__ function:

```go
func (r *HostInfoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
        _ = context.Background()
        _ = r.Log.WithValues("hostinfo", req.NamespacedName)

        // your logic here

        return ctrl.Result{}, nil
}
```

Considering the business logic that I described above, this is what my updated __HostInfoReconciler__ function looks like. Hopefully the comments make is easy to understand, but at the end of the day, when this controller gets a reconcile request (something as simple as a get command will trigger this), the TotalCPU and FreeCPU fields in the status of the Custom Resource are updated for the specific ESXi host in the spec.hostname field. Note that I have omitted a number of required imports that also need to be added to the controller. Refer to the code for the complete [__hostinfo_controller.go__](./controllers/hostinfo_controller.go) code. One thing to note is that I am enabling insecure logins by default. This is something that you may wish to change in your code.

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

        // Share govc's session cache 
        // See https://github.com/vmware/govmomi/blob/master/examples/examples.go

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
        // Print summary for host in HostInfo specification info only
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

With the controller logic now in place, we can now proceed to build the controller / manager.

## Step 7 - Build the controller ##

At this point everything is in place to enable us to deploy the controller to the Kubernete cluster. If you remember back to the prerequisites in step 1, we said that you need access to a container image registry, such as docker.io or quay.io, or VMware's own [Harbor](https://github.com/goharbor/harbor/blob/master/README.md) registry. This is where we need this access to a registry, as we need to push the controller's container image somewhere that can be accessed from your Kubernetes cluster.

The __Dockerfile__ with the appropriate directives is already in place to build the container image and include the controller / manager logic. This was once again taken care of by kubebuilder. You must ensure that you login to your image repository, i.e. docker login, before proceeding with the __make__ commands, e.g.

```shell
$ docker login
Login with your Docker ID to push and pull images from Docker Hub. If you dont have a Docker ID, head over to https://hub.docker.com to create one.
Username: cormachogan
Password: `***********`
WARNING! Your password will be stored unencrypted in /home/cormac/.docker/config.json.
Configure a credential helper to remove this warning. See
https://docs.docker.com/engine/reference/commandline/login/#credentials-store

Login Succeeded
$
```

Next, set an environment variable called __IMG__ to point to your container image repository along with the name and version of the container image, e.g:

```shell
export IMG=docker.io/cormachogan/hostinfo-controller:v1
```

Next, to create the container image of the controller / manager, and push it to the image container repository in a single step, run the following __make__ command. You could of course run this as two seperate commands as well, ```make docker-build``` followed by ```make docker-push``` if you so wished.

```cmd
make docker-build docker-push IMG=docker.io/cormachogan/hostinfo-controller:v1
```

The output has been shortened in this example:

```Makefile
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
. <-- snip!
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

The container image of the controller is now built and pushed to the container image registry. But we have not yet deployed it. We have to do one or two further modifications before we take that step.

## Step 8 - Modify the Manager manifest to include environment variables ##

Kubebuilder provides a manager manifest scaffold file for deploying the controller. However, since we need to provide vCenter details to our controller, we need to add these to the controller/manager manifest file. This is found in __config/manager/manager.yaml__. This manifest contains the deployment for the controller. In the spec, we need to add an additional __spec.env__ section which has the environment variables defined, as well as the name of our __secret__ (which we will create shortly). Below is a snippet of that code. Here is the code-complete [config/manager/manager.yaml](./config/manager/manager.yaml)).

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

Note that the __secret__, called __vc-creds__ above, contains the vCenter credentials. This secret needs to be deployed in the same namespace that the controller is going to run in, which is __hostinfo-system__. Thus, the namespace and secret are created using the following commands, with the environment modified to your own vSphere infrastructure obviously:

```shell
$ kubectl create ns hostinfo-system
namespace/hostinfo-system created
```

```shell
$ kubectl create secret generic vc-creds \
--from-literal='GOVMOMI_USERNAME=administrator@vsphere.local' \
--from-literal='GOVMOMI_PASSWORD=VMware123!' \
--from-literal='GOVMOMI_URL=192.168.0.100' \
-n hostinfo-system
secret/vc-creds created
```

We are now ready to deploy the controller to the Kubernetes cluster.

## Step 9 - Deploy the controller ##

To deploy the controller, we run another __make__ command. This will take care of all of the RBAC, cluster roles and role bindings necessary to run the controller, as well as pinging up the correct image, etc.

```Makefile
make deploy IMG=docker.io/cormachogan/hostinfo-controller:v1
```

The output looks something like this:

```Makefile
$ make deploy IMG=docker.io/cormachogan/hostinfo-controller:v1
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/usr/share/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cd config/manager && kustomize edit set image controller=docker.io/cormachogan/hostinfo-controller:v1
kustomize build config/default | kubectl apply -f -
namespace/hostinfo-system unchanged
customresourcedefinition.apiextensions.k8s.io/hostinfoes.topology.corinternal.com configured
role.rbac.authorization.k8s.io/hostinfo-leader-election-role created
clusterrole.rbac.authorization.k8s.io/hostinfo-manager-role created
clusterrole.rbac.authorization.k8s.io/hostinfo-proxy-role created
clusterrole.rbac.authorization.k8s.io/hostinfo-metrics-reader created
rolebinding.rbac.authorization.k8s.io/hostinfo-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/hostinfo-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/hostinfo-proxy-rolebinding created
service/hostinfo-controller-manager-metrics-service created
deployment.apps/hostinfo-controller-manager created
```

## Step 10 - Check controller functionality ##

Now that our controller has been deployed, let's see if it is working. There are a few different commands that we can run to verify the operator is working.

### Step 10.1 - Check the deployment and replicaset ###

The deployment should be READY. Remember to specify the namespace correctly when checking it.

```shell
$ kubectl get rs -n hostinfo-system
NAME                                     DESIRED   CURRENT   READY   AGE
hostinfo-controller-manager-66bdb8f5bd   1         1         0       9m48s

$ kubectl get deploy -n hostinfo-system
NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
hostinfo-controller-manager   1/1     1            1           14m
```

### Step 10.2 - Check the Pods ###

The deployment manages a single controller Pod. There should be 2 containers READY in the controller Pod. One is the __controller / manager__ and the other is the __kube-rbac-proxy__. The [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy/blob/master/README.md) is a small HTTP proxy that can perform RBAC authorization against the Kubernetes API. It restricts requests to authorized Pods only.

```shell
$ kubectl get pods -n hostinfo-system
NAME                                           READY   STATUS    RESTARTS   AGE
hostinfo-controller-manager-6484c486ff-8vwsn   2/2     Running   0          72s
```

If you experience issues with the one of the pods not coming online, use the following command to display the Pod status and examine the events.

```shell
kubectl describe pod hostinfo-controller-manager-6484c486ff-8vwsn -n hostinfo-system
```

### Step 10.3 - Check the controller / manager logs ###

If we query the __logs__ on the manager container, we should be able to observe successful startup messages as well as successful reconcile requests from the HostInfo CR that we already deployed back in step 5. These reconcile requests should update the __Status__ fields with CPU information as per our controller logic. The command to query the manager container logs in the controller Pod is as follows:

```shell
kubectl logs hostinfo-controller-manager-6484c486ff-8vwsn -n hostinfo-system manager
```

The output should be somewhat similar to this:

```shell
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

Last but not least, let's see if we can see the CPU information in the __status__ fields of the HostInfo object created earlier.

```yaml
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

__Success!!!__ Note that the output above is showing us ```freeCPU``` and ```totalCPU``` as per our business logic implemented in the controller. How cool is that? You can now go ahead and create additional HostInfo manifests for different hosts in your vSphere environment managed by your vCenter server by specifying different hostnames in the manifest spec, and all you to get free and total CPU from those ESXi hosts as well.

## Cleanup ##

To remove the __hostinfo__ CR, operator and CRD, run the following commands.

### Remove the HostInfo CR ###

```shell
$ kubectl delete hostinfo hostinfo-host-e
hostinfo.topology.corinternal.com "hostinfo-host-e" deleted
```

### Removed the Operator/Controller deployment ###

Deleting the deployment will removed the ReplicaSet and Pods associated with the controller.

```shell
$ kubectl get deploy -n hostinfo-system
NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
hostinfo-controller-manager   1/1     1            1           2d8h
```

```shell
$ kubectl delete deploy hostinfo-controller-manager -n hostinfo-system
deployment.apps "hostinfo-controller-manager" deleted
```

### Remove the CRD ###

Next, remove the Custom Resource Definition, __hostinfoes.topology.corinternal.com__.

```shell
$ kubectl get crds
NAME                                                               CREATED AT
antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com        2021-01-14T16:31:58Z
antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com   2021-01-14T16:31:58Z
clusternetworkpolicies.security.antrea.tanzu.vmware.com            2021-01-14T16:31:59Z
hostinfoes.topology.corinternal.com                                2021-01-14T16:52:11Z
traceflows.ops.antrea.tanzu.vmware.com                             2021-01-14T16:31:59Z
```

```Makefile
$ make uninstall
go: creating new go.mod: module tmp
go: found sigs.k8s.io/controller-tools/cmd/controller-gen in sigs.k8s.io/controller-tools v0.2.5
/home/cormac/go/bin/controller-gen "crd:preserveUnknownFields=false,crdVersions=v1,trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
kustomize build config/crd | kubectl delete -f -
customresourcedefinition.apiextensions.k8s.io "hostinfoes.topology.corinternal.com" deleted
```

```shell
$ kubectl get crds
NAME                                                               CREATED AT
antreaagentinfos.clusterinformation.antrea.tanzu.vmware.com        2021-01-14T16:31:58Z
antreacontrollerinfos.clusterinformation.antrea.tanzu.vmware.com   2021-01-14T16:31:58Z
clusternetworkpolicies.security.antrea.tanzu.vmware.com            2021-01-14T16:31:59Z
traceflows.ops.antrea.tanzu.vmware.com                             2021-01-14T16:31:59Z
```

The CRD is now removed. At this point, you can also delete the namespace created for the exercise, in this case __hostinfo-system__. Removing this namespace will also remove the __vc_creds__ secret created earlier.

## What next? ##

One thing you could do it to extend the HostInfo fields and Operator logic so that it returns even more information about the ESXi host. You could add additional Status fields that returned memory, host type, host tags, etc. There is a lot of information that can be retrieved via the govmomi __HostSystem__ API call.

You can now use __kusomtize__ to package the CRD and controller and distribute it to other Kubernetes clusters. Simply point the __kustomize build__ command at the location of the __kustomize.yaml__ file which is in __config/default__.

```shell
kustomize build config/default/ >> /tmp/hostinfo.yaml
```

This newly created __hostinfo.yaml__ manifest includes the CRD, RBAC, Service and Deployment for rolling out the operator on other Kubernetes clusters. Nice, eh?

Finally, if this exercise has given you a desire to do more exciting stuff with Kubernetes Operators when Kubernetes is running on vSphere, check out the [vmGroup](https://github.com/embano1/codeconnect-vm-operator/blob/main/README.md) operator that my colleague __Micheal Gasch__ created. It will let you deploy and manage a set of virtual machines on your vSphere infrastructure via a Kubernetes operator. Cool stuff for sure.
