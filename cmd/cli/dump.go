package cli

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	batchclient "k8s.io/client-go/kubernetes/typed/batch/v1"
	networkingclient "k8s.io/client-go/kubernetes/typed/networking/v1"
	rbacclient "k8s.io/client-go/kubernetes/typed/rbac/v1"
	storageclient "k8s.io/client-go/kubernetes/typed/storage/v1"
	. "k8s.io/kubectl/pkg/cmd/clusterinfo"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"

	// "k8s.io/kubectl/pkg/cmd/plugin"
	// "k8s.io/kubectl/pkg/cmd"
	"io"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultPodLogsTimeout = 20 * time.Second
	timeout               = 5 * time.Minute
)

type ExtraInfoDumpOptions struct {
	NetworkingClient networkingclient.NetworkingV1Interface
	StorageClient    storageclient.StorageV1Interface
	BatchClient      batchclient.BatchV1Interface
	RbacClient       *rbacclient.RbacV1Client
	ClusterInfoDumpOptions
}

var defaultConfigFlags = genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)

func init() {
	// ko := &cmd.KubectlOptions{
	// 	PluginHandler: NewDefaultPluginHandler([]string{"kubedmp"}),
	// 	Arguments:     os.Args,
	// 	ConfigFlags:   defaultConfigFlags,
	// 	IOStreams:     genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
	// }
	// kubeConfigFlags := ko.ConfigFlags
	// if kubeConfigFlags == nil {
	// 	kubeConfigFlags = defaultConfigFlags
	// }
	// flags := rootCmd.PersistentFlags()
	// defaultConfigFlags.AddFlags(flags)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(defaultConfigFlags)
	// matchVersionKubeConfigFlags.AddFlags(flags)
	restClientGetter := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	o := &ExtraInfoDumpOptions{
		ClusterInfoDumpOptions: ClusterInfoDumpOptions{
			PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(scheme.Scheme).WithDefaultOutput("json"),

			IOStreams: genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		},
	}
	dumpCmd := &cobra.Command{
		Use:                   "dump",
		DisableFlagsInUseLine: true,
		Short:                 "Dump relevant information for debugging and diagnosis",
		Long: `
Dump cluster information out suitable for debugging and diagnosing cluster problems.  By default, dumps everything to
stdout. You can optionally specify a directory with --output-directory.  If you specify a directory, Kubernetes will
build a set of files in that directory.  By default, only dumps things in the current namespace and 'kube-system' namespace, but you can
switch to a different namespace with the --namespaces flag, or specify --all-namespaces to dump all namespaces.

The command also dumps the logs of all of the pods in the cluster; these logs are dumped into different directories
based on namespace and pod name.`,
		Example: `
# Dump current cluster state to stdout
kubedmp dump

# Dump current cluster state to /path/to/cluster-state
kubedmp dump --output-directory=/path/to/cluster-state

# Dump all namespaces to stdout
kubedmp dump --all-namespaces

# Dump a set of namespaces to /path/to/cluster-state
kubedmp dump --namespaces default,kube-system --output-directory=/path/to/cluster-state`,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(restClientGetter, cmd))
			cmdutil.CheckErr(o.CompleteExtra(restClientGetter, cmd))
			cmdutil.CheckErr(o.runExtra())
			cmdutil.CheckErr(o.Run())
		},
	}
	// o.PrintFlags.AddFlags(dumpCmd)
	dumpCmd.Flags().StringVar(&o.OutputDir, "output-directory", o.OutputDir, "Where to output the files.  If empty or '-' uses stdout, otherwise creates a directory hierarchy in that directory")
	dumpCmd.Flags().StringSliceVar(&o.Namespaces, "namespaces", o.Namespaces, "A comma separated list of namespaces to dump.")
	dumpCmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", o.AllNamespaces, "If true, dump all namespaces.  If true, --namespaces is ignored.")
	dumpCmd.Flags().StringVar(defaultConfigFlags.KubeConfig, "kubeconfig", *defaultConfigFlags.KubeConfig, "Path to the kubeconfig file to use for CLI requests.")

	cmdutil.AddPodRunningTimeoutFlag(dumpCmd, defaultPodLogsTimeout)
	rootCmd.AddCommand(dumpCmd)
}

func setupOutputWriter(dir string, defaultWriter io.Writer, filename string, fileExtension string) io.Writer {
	if len(dir) == 0 || dir == "-" {
		return defaultWriter
	}
	fullFile := path.Join(dir, filename) + fileExtension
	parent := path.Dir(fullFile)
	cmdutil.CheckErr(os.MkdirAll(parent, 0755))

	file, err := os.Create(fullFile)
	cmdutil.CheckErr(err)
	return file
}

func (o *ExtraInfoDumpOptions) CompleteExtra(restClientGetter genericclioptions.RESTClientGetter, cmd *cobra.Command) error {
	config, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return err
	}
	o.NetworkingClient, err = networkingclient.NewForConfig(config)
	if err != nil {
		return err
	}

	o.StorageClient, err = storageclient.NewForConfig(config)
	if err != nil {
		return err
	}

	o.BatchClient, err = batchclient.NewForConfig(config)
	if err != nil {
		return err
	}

	o.RbacClient, err = rbacclient.NewForConfig(config)
	if err != nil {
		return err
	}

	return nil
}

func (o *ExtraInfoDumpOptions) runExtra() error {

	fileExtension := ".txt"
	if o.PrintFlags.OutputFormat != nil {
		switch *o.PrintFlags.OutputFormat {
		case "json":
			fileExtension = ".json"
		case "yaml":
			fileExtension = ".yaml"
		}
	}

	pvs, err := o.CoreClient.PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if err := o.PrintObj(pvs, setupOutputWriter(o.OutputDir, o.Out, "pvs", fileExtension)); err != nil {
		return err
	}

	scs, err := o.StorageClient.StorageClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if err := o.PrintObj(scs, setupOutputWriter(o.OutputDir, o.Out, "scs", fileExtension)); err != nil {
		return err
	}

	clusterroles, err := o.RbacClient.ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if err := o.PrintObj(clusterroles, setupOutputWriter(o.OutputDir, o.Out, "clusterroles", fileExtension)); err != nil {
		return err
	}

	clusterrolebindings, err := o.RbacClient.ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if err := o.PrintObj(clusterrolebindings, setupOutputWriter(o.OutputDir, o.Out, "clusterrolebindings", fileExtension)); err != nil {
		return err
	}

	var namespaces []string
	if o.AllNamespaces {
		namespaceList, err := o.CoreClient.Namespaces().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		for ix := range namespaceList.Items {
			namespaces = append(namespaces, namespaceList.Items[ix].Name)
		}
	} else {
		if len(o.Namespaces) == 0 {
			namespaces = []string{
				metav1.NamespaceSystem,
				o.Namespace,
			}
		} else {
			namespaces = o.Namespaces
		}
	}
	for _, namespace := range namespaces {
		endpoints, err := o.CoreClient.Endpoints(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(endpoints, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "endpoints"), fileExtension)); err != nil {
			return err
		}

		pvcs, err := o.CoreClient.PersistentVolumeClaims(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(pvcs, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "pvcs"), fileExtension)); err != nil {
			return err
		}

		secrets, err := o.CoreClient.Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(secrets, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "secrets"), fileExtension)); err != nil {
			return err
		}

		configmaps, err := o.CoreClient.ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(configmaps, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "configmaps"), fileExtension)); err != nil {
			return err
		}

		serviceaccounts, err := o.CoreClient.ServiceAccounts(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(serviceaccounts, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "serviceaccounts"), fileExtension)); err != nil {
			return err
		}

		statefulsets, err := o.AppsClient.StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(statefulsets, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "statefulsets"), fileExtension)); err != nil {
			return err
		}

		ingresses, err := o.NetworkingClient.Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(ingresses, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "ingresses"), fileExtension)); err != nil {
			return err
		}

		jobs, err := o.BatchClient.Jobs(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(jobs, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "jobs"), fileExtension)); err != nil {
			return err
		}

		cronjobs, err := o.BatchClient.CronJobs(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(cronjobs, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "cronjobs"), fileExtension)); err != nil {
			return err
		}

		roles, err := o.RbacClient.Roles(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(roles, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "roles"), fileExtension)); err != nil {
			return err
		}

		rolebindings, err := o.RbacClient.RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if err := o.PrintObj(rolebindings, setupOutputWriter(o.OutputDir, o.Out, path.Join(namespace, "rolebindings"), fileExtension)); err != nil {
			return err
		}

	}
	return nil
}
