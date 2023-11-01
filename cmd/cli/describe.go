package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	// "net"
	// "net/url"
	// "os"
	"bytes"
	"io"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	. "k8s.io/kubectl/pkg/describe"
	"k8s.io/kubectl/pkg/util/rbac"
)

var (
	// globally skipped annotations
	skipAnnotations  = sets.NewString(corev1.LastAppliedConfigAnnotation)
	maxAnnotationLen = 140
)

var describeCmd = &cobra.Command{
	Use:                   "describe TYPE RESOURCE_NAME [-n NAMESPACE]",
	DisableFlagsInUseLine: true,
	Short:                 "Show details of a specific resource",
	Long:                  `Show details of a specific resource. Print a detailed description of the selected resource.`,
	Example: `  # Describe a node
  $ kubedmp describe no juju-ceba75-k8s-2
  
  # Describe a pod in kube-system namespace
  $ kubedmp describe po coredns-6bcf44f4cc-j9wkq -n kube-system`,
	// Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 2 {
			log.Fatalf("Please specify a type and an object name\n")
			return
		}
		resType = args[0]
		resName = args[1]

		if !hasType(resType) {
			return
		}
		filePath := dumpFile
		if len(dumpDir) > 0 {
			filename := ""
			switch resType {
			case "no", "node":
				filename = "nodes"
				resNamespace = ""
			case "po", "pod":
				filename = "pods"
			case "svc", "service":
				filename = "services"
			case "deploy", "deployment":
				filename = "deployments"
			case "ds", "daemonset":
				filename = "daemonsets"
			case "rs", "replicaset":
				filename = "replicasets"
			case "sts", "statefulset":
				filename = "statefulsets"
			case "pvc", "persistentvolumeclaim":
				filename = "pvcs"
			case "event":
				filename = "events"
			case "pv", "persistentvolume":
				filename = "pvs"
				resNamespace = ""
			case "cm", "configmap", "configmaps":
				filename = "configmaps"
			case "secret", "secrets":
				filename = "secrets"
			case "sa", "serviceaccount", "serviceaccounts":
				filename = "serviceaccounts"
			case "ing", "ingress", "ingresses":
				filename = "ingresses"
			case "sc", "storageclass", "storageclasses":
				filename = "scs"
				resNamespace = ""
			case "clusterrole", "clusterroles":
				filename = "clusterroles"
				resNamespace = ""
			case "clusterrolebinding", "clusterrolebindings":
				filename = "clusterrolebindings"
				resNamespace = ""
			case "ep", "endpoint", "endpoints":
				filename = "endpoints"
			case "job", "jobs":
				filename = "jobs"
			case "cj", "cronjob", "cronjobs":
				filename = "cronjobs"
			case "role", "roles":
				filename = "roles"
			case "rolebinding", "rolebindings":
				filename = "rolebindings"
			}
			filePath = filepath.Join(dumpDir, resNamespace, filename+"."+dumpFormat)
		}

		readFile(filePath, describeObject)

	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
	describeCmd.Flags().StringVarP(&resNamespace, ns, "n", "default", "namespace of the resource, not applicable to node")
	describeCmd.PersistentFlags().StringVarP(&dumpFile, dumpFileFlag, "f", "./cluster-info.dump", "Path to dump file")
	describeCmd.PersistentFlags().StringVarP(&dumpDir, dumpDirFlag, "d", "", "Path to dump directory")

}

func tabbedString(f func(io.Writer) error) (string, error) {
	out := new(tabwriter.Writer)
	buf := &bytes.Buffer{}
	out.Init(buf, 0, 8, 2, ' ', 0)

	err := f(out)
	if err != nil {
		return "", err
	}

	out.Flush()
	return buf.String(), nil
}

func describeObject(buffer string) {
	var result map[string]interface{}
	// fmt.Println(buffer)
	// fmt.Println(resType, resNamespace, resName)
	err := json.Unmarshal([]byte(buffer), &result)

	if err != nil {
		// log.Fatalf("Error processing buffer: %v\n%v\n", err.Error(), buffer)
		return
	}
	// fmt.Println(result["kind"].(string))
	if kind, ok := result["kind"].(string); ok {

		if inType(resType, "Node") && kind == "NodeList" {
			var nodeList corev1.NodeList
			err := json.Unmarshal([]byte(buffer), &nodeList)
			if err != nil {
				log.Fatalf("Error parsing node list: %v\n%v\n", err.Error(), buffer)
			}
			for _, node := range nodeList.Items {
				// fmt.Println("Checking ", node.Name)
				if resName == node.Name {

					s, err := DefaultObjectDescriber.DescribeObject(&node)
					if err != nil {
						log.Fatalf("Error generating output for node %s: %s", node.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Persistent Volume") && kind == "PersistentVolumeList" {
			var pvList corev1.PersistentVolumeList
			err := json.Unmarshal([]byte(buffer), &pvList)
			if err != nil {
				log.Fatalf("Error parsing pv list: %v\n%v\n", err.Error(), buffer)
			}
			for _, pv := range pvList.Items {
				if resName == pv.Name {
					s, err := DefaultObjectDescriber.DescribeObject(&pv)
					if err != nil {
						log.Fatalf("Error generating output for pv %s: %s", pv.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Storage Class") && kind == "StorageClassList" {
			var scList storagev1.StorageClassList
			err := json.Unmarshal([]byte(buffer), &scList)
			if err != nil {
				log.Fatalf("Error parsing sc list: %v\n%v\n", err.Error(), buffer)
			}
			for _, sc := range scList.Items {
				if resName == sc.Name {
					s, err := DefaultObjectDescriber.DescribeObject(&sc)
					if err != nil {
						log.Fatalf("Error generating output for sc %s: %s", sc.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Cluster Role") && kind == "ClusterRoleList" {
			var crList rbacv1.ClusterRoleList
			err := json.Unmarshal([]byte(buffer), &crList)
			if err != nil {
				log.Fatalf("Error parsing cluster role list: %v\n%v\n", err.Error(), buffer)
			}
			for _, cr := range crList.Items {
				if resName == cr.Name {
					s, err := describeClusterRole(&cr)
					if err != nil {
						log.Fatalf("Error generating output for cluster role %s: %s", cr.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Cluster Role Binding") && kind == "ClusterRoleBindingList" {
			var crbList rbacv1.ClusterRoleBindingList
			err := json.Unmarshal([]byte(buffer), &crbList)
			if err != nil {
				log.Fatalf("Error parsing cluster role binding list: %v\n%v\n", err.Error(), buffer)
			}
			for _, crb := range crbList.Items {
				if resName == crb.Name {
					s, err := describeClusterRoleBinding(&crb)
					if err != nil {
						log.Fatalf("Error generating output for cluster role binding %s: %s", crb.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Pod") && kind == "PodList" {
			var podList corev1.PodList
			err := json.Unmarshal([]byte(buffer), &podList)
			if err != nil {
				log.Fatalf("Error parsing pod list: %v\n%v\n", err.Error(), buffer)
			}
			for _, pod := range podList.Items {
				if resName == pod.Name && resNamespace == pod.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&pod)
					if err != nil {
						log.Fatalf("Error generating output for pod %s/%s: %s", pod.Namespace, pod.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Service") && kind == "ServiceList" {
			var svcList corev1.ServiceList
			err := json.Unmarshal([]byte(buffer), &svcList)
			if err != nil {
				log.Fatalf("Error parsing service list: %v\n%v\n", err.Error(), buffer)
			}
			for _, svc := range svcList.Items {
				if resName == svc.Name && resNamespace == svc.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&svc)
					if err != nil {
						log.Fatalf("Error generating output for service %s/%s: %s", svc.Namespace, svc.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Deployment") && kind == "DeploymentList" {
			var deployList appsv1.DeploymentList
			err := json.Unmarshal([]byte(buffer), &deployList)
			if err != nil {
				log.Fatalf("Error parsing deployment list: %v\n%v\n", err.Error(), buffer)
			}
			for _, deploy := range deployList.Items {
				if resName == deploy.Name && resNamespace == deploy.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&deploy)
					if err != nil {
						log.Fatalf("Error generating output for deployment %s/%s: %s", deploy.Namespace, deploy.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "DaemonSet") && kind == "DaemonSetList" {
			var daemonList appsv1.DaemonSetList
			err := json.Unmarshal([]byte(buffer), &daemonList)
			if err != nil {
				log.Fatalf("Error parsing daemonset list: %v\n%v\n", err.Error(), buffer)
			}
			for _, daemon := range daemonList.Items {
				if resName == daemon.Name && resNamespace == daemon.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&daemon)
					if err != nil {
						log.Fatalf("Error generating output for daemonset %s/%s: %s", daemon.Namespace, daemon.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "ReplicaSet") && kind == "ReplicaSetList" {
			var replicaList appsv1.ReplicaSetList
			err := json.Unmarshal([]byte(buffer), &replicaList)
			if err != nil {
				log.Fatalf("Error parsing replicaset list: %v\n%v\n", err.Error(), buffer)
			}
			for _, rs := range replicaList.Items {
				if resName == rs.Name && resNamespace == rs.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&rs)
					if err != nil {
						log.Fatalf("Error generating output for replicaset %s/%s: %s", rs.Namespace, rs.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "StatefulSet") && kind == "StatefulSetList" {
			var stsList appsv1.StatefulSetList
			err := json.Unmarshal([]byte(buffer), &stsList)
			if err != nil {
				log.Fatalf("Error parsing statefulset list: %v\n%v\n", err.Error(), buffer)
			}
			for _, sts := range stsList.Items {
				if resName == sts.Name && resNamespace == sts.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&sts)
					if err != nil {
						log.Fatalf("Error generating output for statefulset %s/%s: %s", sts.Namespace, sts.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Persistent Volume Claim") && kind == "PersistentVolumeClaimList" {
			var pvcList corev1.PersistentVolumeClaimList
			err := json.Unmarshal([]byte(buffer), &pvcList)
			if err != nil {
				log.Fatalf("Error parsing pvc list: %v\n%v\n", err.Error(), buffer)
			}
			for _, pvc := range pvcList.Items {
				if resName == pvc.Name && resNamespace == pvc.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&pvc)
					if err != nil {
						log.Fatalf("Error generating output for pvc %s/%s: %s", pvc.Namespace, pvc.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "ConfigMap") && kind == "ConfigMapList" {
			var cmList corev1.ConfigMapList
			err := json.Unmarshal([]byte(buffer), &cmList)
			if err != nil {
				log.Fatalf("Error parsing cm list: %v\n%v\n", err.Error(), buffer)
			}
			for _, cm := range cmList.Items {
				if resName == cm.Name && resNamespace == cm.Namespace {
					s, err := describeConfigMap(&cm)
					if err != nil {
						log.Fatalf("Error generating output for cm %s/%s: %s", cm.Namespace, cm.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Secret") && kind == "SecretList" {
			var secretList corev1.SecretList
			err := json.Unmarshal([]byte(buffer), &secretList)
			if err != nil {
				log.Fatalf("Error parsing secret list: %v\n%v\n", err.Error(), buffer)
			}
			for _, scrt := range secretList.Items {
				if resName == scrt.Name && resNamespace == scrt.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&scrt)
					if err != nil {
						log.Fatalf("Error generating output for secret %s/%s: %s", scrt.Namespace, scrt.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Service Account") && kind == "ServiceAccountList" {
			var saList corev1.ServiceAccountList
			err := json.Unmarshal([]byte(buffer), &saList)
			if err != nil {
				log.Fatalf("Error parsing sa list: %v\n%v\n", err.Error(), buffer)
			}
			for _, sa := range saList.Items {
				if resName == sa.Name && resNamespace == sa.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&sa)
					if err != nil {
						log.Fatalf("Error generating output for sa %s/%s: %s", sa.Namespace, sa.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Ingress") && kind == "IngressList" {
			var ingList networkingv1.IngressList
			err := json.Unmarshal([]byte(buffer), &ingList)
			if err != nil {
				log.Fatalf("Error parsing ingress list: %v\n%v\n", err.Error(), buffer)
			}
			for _, ing := range ingList.Items {
				if resName == ing.Name && resNamespace == ing.Namespace {
					s, err := describeIngressV1(&ing)
					if err != nil {
						log.Fatalf("Error generating output for ingress %s/%s: %s", ing.Namespace, ing.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Endpoints") && kind == "EndpointsList" {
			var epList corev1.EndpointsList
			err := json.Unmarshal([]byte(buffer), &epList)
			if err != nil {
				log.Fatalf("Error parsing ep list: %v\n%v\n", err.Error(), buffer)
			}
			for _, ep := range epList.Items {
				if resName == ep.Name && resNamespace == ep.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&ep)
					if err != nil {
						log.Fatalf("Error generating output for ep %s/%s: %s", ep.Namespace, ep.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Job") && kind == "JobList" {
			var jobList batchv1.JobList
			err := json.Unmarshal([]byte(buffer), &jobList)
			if err != nil {
				log.Fatalf("Error parsing job list: %v\n%v\n", err.Error(), buffer)
			}
			for _, job := range jobList.Items {
				if resName == job.Name && resNamespace == job.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&job)
					if err != nil {
						log.Fatalf("Error generating output for job %s/%s: %s", job.Namespace, job.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Cron Job") && kind == "CronJobList" {
			var jobList batchv1.CronJobList
			err := json.Unmarshal([]byte(buffer), &jobList)
			if err != nil {
				log.Fatalf("Error parsing cron job list: %v\n%v\n", err.Error(), buffer)
			}
			for _, job := range jobList.Items {
				if resName == job.Name && resNamespace == job.Namespace {
					s, err := DefaultObjectDescriber.DescribeObject(&job)
					if err != nil {
						log.Fatalf("Error generating output for cron job %s/%s: %s", job.Namespace, job.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Role") && kind == "RoleList" {
			var roleList rbacv1.RoleList
			err := json.Unmarshal([]byte(buffer), &roleList)
			if err != nil {
				log.Fatalf("Error parsing role list: %v\n%v\n", err.Error(), buffer)
			}
			for _, role := range roleList.Items {
				if resName == role.Name {
					s, err := describeRole(&role)
					if err != nil {
						log.Fatalf("Error generating output for role %s: %s", role.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if inType(resType, "Role Binding") && kind == "RoleBindingList" {
			var rbList rbacv1.RoleBindingList
			err := json.Unmarshal([]byte(buffer), &rbList)
			if err != nil {
				log.Fatalf("Error parsing role binding list: %v\n%v\n", err.Error(), buffer)
			}
			for _, rb := range rbList.Items {
				if resName == rb.Name {
					s, err := describeRoleBinding(&rb)
					if err != nil {
						log.Fatalf("Error generating output for role binding %s: %s", rb.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		}

	}
}

func describeRoleBinding(binding *rbacv1.RoleBinding) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", binding.Name)
		printLabelsMultiline(w, "Labels", binding.Labels)
		printAnnotationsMultiline(w, "Annotations", binding.Annotations)

		w.Write(LEVEL_0, "Role:\n")
		w.Write(LEVEL_1, "Kind:\t%s\n", binding.RoleRef.Kind)
		w.Write(LEVEL_1, "Name:\t%s\n", binding.RoleRef.Name)

		w.Write(LEVEL_0, "Subjects:\n")
		w.Write(LEVEL_1, "Kind\tName\tNamespace\n")
		w.Write(LEVEL_1, "----\t----\t---------\n")
		for _, s := range binding.Subjects {
			w.Write(LEVEL_1, "%s\t%s\t%s\n", s.Kind, s.Name, s.Namespace)
		}

		return nil
	})
}

func describeClusterRoleBinding(binding *rbacv1.ClusterRoleBinding) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", binding.Name)
		printLabelsMultiline(w, "Labels", binding.Labels)
		printAnnotationsMultiline(w, "Annotations", binding.Annotations)

		w.Write(LEVEL_0, "Role:\n")
		w.Write(LEVEL_1, "Kind:\t%s\n", binding.RoleRef.Kind)
		w.Write(LEVEL_1, "Name:\t%s\n", binding.RoleRef.Name)

		w.Write(LEVEL_0, "Subjects:\n")
		w.Write(LEVEL_1, "Kind\tName\tNamespace\n")
		w.Write(LEVEL_1, "----\t----\t---------\n")
		for _, s := range binding.Subjects {
			w.Write(LEVEL_1, "%s\t%s\t%s\n", s.Kind, s.Name, s.Namespace)
		}

		return nil
	})
}

func describeRole(role *rbacv1.Role) (string, error) {
	breakdownRules := []rbacv1.PolicyRule{}
	for _, rule := range role.Rules {
		breakdownRules = append(breakdownRules, rbac.BreakdownRule(rule)...)
	}

	compactRules, err := rbac.CompactRules(breakdownRules)
	if err != nil {
		return "", err
	}
	sort.Stable(rbac.SortableRuleSlice(compactRules))

	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", role.Name)
		printLabelsMultiline(w, "Labels", role.Labels)
		printAnnotationsMultiline(w, "Annotations", role.Annotations)

		w.Write(LEVEL_0, "PolicyRule:\n")
		w.Write(LEVEL_1, "Resources\tNon-Resource URLs\tResource Names\tVerbs\n")
		w.Write(LEVEL_1, "---------\t-----------------\t--------------\t-----\n")
		for _, r := range compactRules {
			w.Write(LEVEL_1, "%s\t%v\t%v\t%v\n", CombineResourceGroup(r.Resources, r.APIGroups), r.NonResourceURLs, r.ResourceNames, r.Verbs)
		}

		return nil
	})
}

func describeClusterRole(role *rbacv1.ClusterRole) (string, error) {
	breakdownRules := []rbacv1.PolicyRule{}
	for _, rule := range role.Rules {
		breakdownRules = append(breakdownRules, rbac.BreakdownRule(rule)...)
	}

	compactRules, err := rbac.CompactRules(breakdownRules)
	if err != nil {
		return "", err
	}
	sort.Stable(rbac.SortableRuleSlice(compactRules))

	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", role.Name)
		printLabelsMultiline(w, "Labels", role.Labels)
		printAnnotationsMultiline(w, "Annotations", role.Annotations)

		w.Write(LEVEL_0, "PolicyRule:\n")
		w.Write(LEVEL_1, "Resources\tNon-Resource URLs\tResource Names\tVerbs\n")
		w.Write(LEVEL_1, "---------\t-----------------\t--------------\t-----\n")
		for _, r := range compactRules {
			w.Write(LEVEL_1, "%s\t%v\t%v\t%v\n", CombineResourceGroup(r.Resources, r.APIGroups), r.NonResourceURLs, r.ResourceNames, r.Verbs)
		}

		return nil
	})
}

func describeConfigMap(configMap *corev1.ConfigMap) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", configMap.Name)
		w.Write(LEVEL_0, "Namespace:\t%s\n", configMap.Namespace)
		printLabelsMultiline(w, "Labels", configMap.Labels)
		printAnnotationsMultiline(w, "Annotations", configMap.Annotations)

		w.Write(LEVEL_0, "\nData\n====\n")
		for k, v := range configMap.Data {
			w.Write(LEVEL_0, "%s:\n----\n", k)
			w.Write(LEVEL_0, "%s\n", string(v))
		}
		w.Write(LEVEL_0, "\nBinaryData\n====\n")
		for k, v := range configMap.BinaryData {
			w.Write(LEVEL_0, "%s: %s bytes\n", k, strconv.Itoa(len(v)))
		}
		w.Write(LEVEL_0, "\n")

		return nil
	})
}

// printLabelsMultiline prints multiple labels with a proper alignment.
func printLabelsMultiline(w PrefixWriter, title string, labels map[string]string) {
	printLabelsMultilineWithIndent(w, "", title, "\t", labels, sets.NewString())
}

// printLabelsMultiline prints multiple labels with a user-defined alignment.
func printLabelsMultilineWithIndent(w PrefixWriter, initialIndent, title, innerIndent string, labels map[string]string, skip sets.String) {
	w.Write(LEVEL_0, "%s%s:%s", initialIndent, title, innerIndent)

	if len(labels) == 0 {
		w.WriteLine("<none>")
		return
	}

	// to print labels in the sorted order
	keys := make([]string, 0, len(labels))
	for key := range labels {
		if skip.Has(key) {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		w.WriteLine("<none>")
		return
	}
	sort.Strings(keys)

	for i, key := range keys {
		if i != 0 {
			w.Write(LEVEL_0, "%s", initialIndent)
			w.Write(LEVEL_0, "%s", innerIndent)
		}
		w.Write(LEVEL_0, "%s=%s\n", key, labels[key])
	}
}

// printAnnotationsMultiline prints multiple annotations with a proper alignment.
// If annotation string is too long, we omit chars more than 200 length.
func printAnnotationsMultiline(w PrefixWriter, title string, annotations map[string]string) {
	w.Write(LEVEL_0, "%s:\t", title)

	// to print labels in the sorted order
	keys := make([]string, 0, len(annotations))
	for key := range annotations {
		if skipAnnotations.Has(key) {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		w.WriteLine("<none>")
		return
	}
	sort.Strings(keys)
	indent := "\t"
	for i, key := range keys {
		if i != 0 {
			w.Write(LEVEL_0, indent)
		}
		value := strings.TrimSuffix(annotations[key], "\n")
		if (len(value)+len(key)+2) > maxAnnotationLen || strings.Contains(value, "\n") {
			w.Write(LEVEL_0, "%s:\n", key)
			for _, s := range strings.Split(value, "\n") {
				w.Write(LEVEL_0, "%s  %s\n", indent, shorten(s, maxAnnotationLen-2))
			}
		} else {
			w.Write(LEVEL_0, "%s: %s\n", key, value)
		}
	}
}

func shorten(s string, maxLength int) string {
	if len(s) > maxLength {
		return s[:maxLength] + "..."
	}
	return s
}

func describeIngressV1(ing *networkingv1.Ingress) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%v\n", ing.Name)
		printLabelsMultiline(w, "Labels", ing.Labels)
		w.Write(LEVEL_0, "Namespace:\t%v\n", ing.Namespace)
		w.Write(LEVEL_0, "Address:\t%v\n", ingressLoadBalancerStatusStringerV1(ing.Status.LoadBalancer, true))
		ingressClassName := "<none>"
		if ing.Spec.IngressClassName != nil {
			ingressClassName = *ing.Spec.IngressClassName
		}
		w.Write(LEVEL_0, "Ingress Class:\t%v\n", ingressClassName)
		def := ing.Spec.DefaultBackend
		ns := ing.Namespace
		defaultBackendDescribe := "<default>"
		if def != nil {
			defaultBackendDescribe = describeBackendV1(ns, def)
		}
		w.Write(LEVEL_0, "Default backend:\t%s\n", defaultBackendDescribe)
		if len(ing.Spec.TLS) != 0 {
			describeIngressTLSV1(w, ing.Spec.TLS)
		}
		w.Write(LEVEL_0, "Rules:\n  Host\tPath\tBackends\n")
		w.Write(LEVEL_1, "----\t----\t--------\n")
		count := 0
		for _, rules := range ing.Spec.Rules {

			if rules.HTTP == nil {
				continue
			}
			count++
			host := rules.Host
			if len(host) == 0 {
				host = "*"
			}
			w.Write(LEVEL_1, "%s\t\n", host)
			for _, path := range rules.HTTP.Paths {
				w.Write(LEVEL_2, "\t%s \t%s\n", path.Path, describeBackendV1(ing.Namespace, &path.Backend))
			}
		}
		if count == 0 {
			w.Write(LEVEL_1, "%s\t%s\t%s\n", "*", "*", defaultBackendDescribe)
		}
		printAnnotationsMultiline(w, "Annotations", ing.Annotations)

		return nil
	})
}

func describeBackendV1(ns string, backend *networkingv1.IngressBackend) string {

	if backend.Service != nil {
		sb := serviceBackendStringer(backend.Service)
		// endpoints, err := i.client.CoreV1().Endpoints(ns).Get(context.TODO(), backend.Service.Name, metav1.GetOptions{})
		// if err != nil {
		// 	return fmt.Sprintf("%v (<error: %v>)", sb, err)
		// }
		// service, err := i.client.CoreV1().Services(ns).Get(context.TODO(), backend.Service.Name, metav1.GetOptions{})
		// if err != nil {
		// 	return fmt.Sprintf("%v(<error: %v>)", sb, err)
		// }
		// spName := ""
		// for i := range service.Spec.Ports {
		// 	sp := &service.Spec.Ports[i]
		// 	if backend.Service.Port.Number != 0 && backend.Service.Port.Number == sp.Port {
		// 		spName = sp.Name
		// 	} else if len(backend.Service.Port.Name) > 0 && backend.Service.Port.Name == sp.Name {
		// 		spName = sp.Name
		// 	}
		// }
		// ep := formatEndpoints(endpoints, sets.NewString(spName))
		return fmt.Sprintf("%s", sb)
	}
	if backend.Resource != nil {
		ic := backend.Resource
		apiGroup := "<none>"
		if ic.APIGroup != nil {
			apiGroup = fmt.Sprintf("%v", *ic.APIGroup)
		}
		return fmt.Sprintf("APIGroup: %v, Kind: %v, Name: %v", apiGroup, ic.Kind, ic.Name)
	}
	return ""
}

// backendStringer behaves just like a string interface and converts the given backend to a string.
func serviceBackendStringer(backend *networkingv1.IngressServiceBackend) string {
	if backend == nil {
		return ""
	}
	var bPort string
	if backend.Port.Number != 0 {
		sNum := int64(backend.Port.Number)
		bPort = strconv.FormatInt(sNum, 10)
	} else {
		bPort = backend.Port.Name
	}
	return fmt.Sprintf("%v:%v", backend.Name, bPort)
}

// ingressLoadBalancerStatusStringerV1 behaves mostly like a string interface and converts the given status to a string.
// `wide` indicates whether the returned value is meant for --o=wide output. If not, it's clipped to 16 bytes.
func ingressLoadBalancerStatusStringerV1(s networkingv1.IngressLoadBalancerStatus, wide bool) string {
	ingress := s.Ingress
	result := sets.NewString()
	for i := range ingress {
		if ingress[i].IP != "" {
			result.Insert(ingress[i].IP)
		} else if ingress[i].Hostname != "" {
			result.Insert(ingress[i].Hostname)
		}
	}

	r := strings.Join(result.List(), ",")
	if !wide && len(r) > LoadBalancerWidth {
		r = r[0:(LoadBalancerWidth-3)] + "..."
	}
	return r
}

func describeIngressTLSV1(w PrefixWriter, ingTLS []networkingv1.IngressTLS) {
	w.Write(LEVEL_0, "TLS:\n")
	for _, t := range ingTLS {
		if t.SecretName == "" {
			w.Write(LEVEL_1, "SNI routes %v\n", strings.Join(t.Hosts, ","))
		} else {
			w.Write(LEVEL_1, "%v terminates %v\n", t.SecretName, strings.Join(t.Hosts, ","))
		}
	}
}
