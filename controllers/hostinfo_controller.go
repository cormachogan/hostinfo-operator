/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	topologyv1 "hostinfo/api/v1"
)

// HostInfoReconciler reconciles a HostInfo object
type HostInfoReconciler struct {
	client.Client
	VC     *vim25.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=topology.corinternal.com,resources=hostinfoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=topology.corinternal.com,resources=hostinfoes/status,verbs=get;update;patch

func (r *HostInfoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	ctx := context.Background()
	log := r.Log.WithValues("hostinfo", req.NamespacedName)

	hi := &topologyv1.HostInfo{}
	if err := r.Client.Get(ctx, req.NamespacedName, hi); err != nil {
		// add some debug information if it's not a NotFound error
		if !k8serr.IsNotFound(err) {
			log.Error(err, "unable to fetch HostInfo")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	msg := fmt.Sprintf("received reconcile request for %q (namespace: %q)", hi.GetName(), hi.GetNamespace())
	log.Info(msg)

	//
	// Create a view manager
	//

	m := view.NewManager(r.VC)

	//
	// Create a container view of HostSystem objects
	//

	v, err := m.CreateContainerView(ctx, r.VC.ServiceContent.RootFolder, []string{"HostSystem"}, true)

	if err != nil {
		msg := fmt.Sprintf("unable to create container view for HostSystem: error %s", err)
		log.Info(msg)
		return ctrl.Result{}, err
	}

	defer v.Destroy(ctx)

	//
	// Retrieve summary property for all hosts
	// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.HostSystem.html
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
		if hs.Summary.Config.Name == hi.Spec.Hostname {
			hi.Status.TotalCPU = int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
			hi.Status.FreeCPU = (int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)) - int64(hs.Summary.QuickStats.OverallCpuUsage)
		}
	}

	//
	// Update the HostInfo status fields
	//

	if err := r.Status().Update(ctx, hi); err != nil {
		log.Error(err, "unable to update HostInfo status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *HostInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&topologyv1.HostInfo{}).
		Complete(r)
}
