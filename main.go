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

package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/vmware/govmomi/session/cache"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"

	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	topologyv1 "hostinfo/api/v1"
	"hostinfo/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = topologyv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func vlogin(ctx context.Context, vc, user, pwd string) (*vim25.Client, error) {

	//
	// Create a vSphere/vCenter client
	//
	//    The govmomi client requires a URL object, u, not just a string representation of the vCenter URL.
	//

	u, err := soap.ParseURL(vc)

	if u == nil {
		fmt.Println("could not parse URL (environment variables set?)")
	}

	if err != nil {
		setupLog.Error(err, "URL parsing not successful", "controller", "HostInfo")
		os.Exit(1)
	}

	u.User = url.UserPassword(user, pwd)

	//
	// Ripped from https://github.com/vmware/govmomi/blob/master/examples/examples.go
	//

	// Share govc's session cache
	s := &cache.Session{
		URL:      u,
		Insecure: true,
	}

	c := new(vim25.Client)

	err = s.Login(ctx, c, nil)

	if err != nil {
		setupLog.Error(err, " login not successful", "controller", "HostInfo")
		os.Exit(1)
	}

	return c, nil
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "0df5945b.corinternal.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	//
	// Retrieve vSphere login detail from environemnt variables
	//

	vc := os.Getenv("GOVMOMI_URL")
	user := os.Getenv("GOVMOMI_USERNAME")
	pwd := os.Getenv("GOVMOMI_PASSWORD")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//
	// Get vSphere login session, and send to Reconciler
	//
	c, err := vlogin(ctx, vc, user, pwd)

	if err = (&controllers.HostInfoReconciler{
		Client: mgr.GetClient(),
		VC:     c,
		Log:    ctrl.Log.WithName("controllers").WithName("HostInfo"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HostInfo")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
