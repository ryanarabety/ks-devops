/*
Copyright 2020 KubeSphere Authors

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

package options

import (
	"flag"
	"strings"
	"time"

	"kubesphere.io/devops/pkg/config"

	"kubesphere.io/devops/pkg/client/devops/jenkins"
	"kubesphere.io/devops/pkg/client/k8s"
	"kubesphere.io/devops/pkg/client/s3"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/leaderelection"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
)

type DevOpsControllerManagerOptions struct {
	KubernetesOptions *k8s.KubernetesOptions
	JenkinsOptions    *jenkins.Options
	LeaderElect       bool
	LeaderElection    *leaderelection.LeaderElectionConfig
	WebhookCertDir    string
	S3Options         *s3.Options
	FeatureOptions    *FeatureOptions
	JWTOptions        *JWTOptions
	ArgoCDOption      *config.ArgoCDOption

	// KubeSphere is using sigs.k8s.io/application as fundamental object to implement Application Management.
	// There are other projects also built on sigs.k8s.io/application, when KubeSphere installed along side
	// them, conflicts happen. So we leave an option to only reconcile applications  matched with the given
	// selector. Default will reconcile all applications.
	//    For example
	//      "kubesphere.io/creator=" means reconcile applications with this label key
	//      "!kubesphere.io/creator" means exclude applications with this key
	ApplicationSelector string
}

func NewDevOpsControllerManagerOptions() *DevOpsControllerManagerOptions {
	s := &DevOpsControllerManagerOptions{
		JenkinsOptions: jenkins.NewJenkinsOptions(),
		LeaderElection: &leaderelection.LeaderElectionConfig{
			LeaseDuration: 30 * time.Second,
			RenewDeadline: 15 * time.Second,
			RetryPeriod:   5 * time.Second,
		},
		FeatureOptions:      NewFeatureOptions(),
		LeaderElect:         false,
		WebhookCertDir:      "",
		ApplicationSelector: "",
		KubernetesOptions:   &k8s.KubernetesOptions{},
		ArgoCDOption:        &config.ArgoCDOption{},
	}

	return s
}

func (s *DevOpsControllerManagerOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	s.KubernetesOptions.AddFlags(fss.FlagSet("kubernetes"), s.KubernetesOptions)
	s.JenkinsOptions.AddFlags(fss.FlagSet("devops"), s.JenkinsOptions)
	s.FeatureOptions.AddFlags(fss.FlagSet("feature"), s.FeatureOptions)
	s.ArgoCDOption.AddFlags(fss.FlagSet("argocd"))

	fs := fss.FlagSet("leaderelection")
	s.bindLeaderElectionFlags(s.LeaderElection, fs)

	fs.BoolVar(&s.LeaderElect, "leader-elect", s.LeaderElect, ""+
		"Whether to enable leader election. This field should be enabled when controller manager"+
		"deployed with multiple replicas.")

	fs.StringVar(&s.WebhookCertDir, "webhook-cert-dir", s.WebhookCertDir, ""+
		"Certificate directory used to setup webhooks, need tls.crt and tls.key placed inside."+
		"if not set, webhook server would look up the server key and certificate in"+
		"{TempDir}/k8s-webhook-server/serving-certs")

	gfs := fss.FlagSet("generic")
	gfs.StringVar(&s.ApplicationSelector, "application-selector", s.ApplicationSelector, ""+
		"Only reconcile application(sigs.k8s.io/application) objects match given selector, this could avoid conflicts with "+
		"other projects built on top of sig-application. Default behavior is to reconcile all of application objects.")

	kfs := fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		kfs.AddGoFlag(fl)
	})

	return fss
}

func (s *DevOpsControllerManagerOptions) Validate() []error {
	var errs []error
	errs = append(errs, s.JenkinsOptions.Validate()...)
	errs = append(errs, s.KubernetesOptions.Validate()...)
	errs = append(errs, s.FeatureOptions.Validate()...)

	if len(s.ApplicationSelector) != 0 {
		_, err := labels.Parse(s.ApplicationSelector)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (s *DevOpsControllerManagerOptions) bindLeaderElectionFlags(l *leaderelection.LeaderElectionConfig, fs *pflag.FlagSet) {
	fs.DurationVar(&l.LeaseDuration, "leader-elect-lease-duration", l.LeaseDuration, ""+
		"The duration that non-leader candidates will wait after observing a leadership "+
		"renewal until attempting to acquire leadership of a led but unrenewed leader "+
		"slot. This is effectively the maximum duration that a leader can be stopped "+
		"before it is replaced by another candidate. This is only applicable if leader "+
		"election is enabled.")
	fs.DurationVar(&l.RenewDeadline, "leader-elect-renew-deadline", l.RenewDeadline, ""+
		"The interval between attempts by the acting master to renew a leadership slot "+
		"before it stops leading. This must be less than or equal to the lease duration. "+
		"This is only applicable if leader election is enabled.")
	fs.DurationVar(&l.RetryPeriod, "leader-elect-retry-period", l.RetryPeriod, ""+
		"The duration the clients should wait between attempting acquisition and renewal "+
		"of a leadership. This is only applicable if leader election is enabled.")
}
