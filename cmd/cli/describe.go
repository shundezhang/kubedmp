package cli

import (
	"encoding/json"
	"fmt"
	"log"

	// "net"
	// "net/url"
	// "os"
	"bytes"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	. "k8s.io/kubectl/pkg/describe"
	"k8s.io/kubectl/pkg/util/qos"
	resourcehelper "k8s.io/kubectl/pkg/util/resource"
	storageutil "k8s.io/kubectl/pkg/util/storage"
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
	Long: `Show details of a specific resource. Print a detailed description of the selected resource.
It can only show detais of one resource, whose type is either node/no, pod/po, service/svc, deployment/deploy, daemonset/ds,
replicaset/rs, persistentvolume/pv, persistentvolumeclaim/pvc, statefulset/sts.`,
	Example: `  # Describe a node
  $ kubedmp describe no juju-ceba75-k8s-2
  
  # Describe a pod in kube-system namespace
  $ kubedmp describe po coredns-6bcf44f4cc-j9wkq -n kube-system`,
	// Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 2 {
			log.Fatalf("Please specify a type, e.g. node/no, pod/po, service/svc, deployment/deploy, daemonset/ds, replicaset/rs, persistentvolume/pv, persistentvolumeclaim/pvc, statefulset/sts and an object name\n")
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
	if kind, ok := result["kind"].(string); ok {

		if (resType == "no" || resType == "node") && kind == "NodeList" {
			var nodeList corev1.NodeList
			err := json.Unmarshal([]byte(buffer), &nodeList)
			if err != nil {
				log.Fatalf("Error parsing node list: %v\n%v\n", err.Error(), buffer)
			}
			for _, node := range nodeList.Items {
				if resName == node.Name {
					s, err := describeNode(node)
					if err != nil {
						log.Fatalf("Error generating output for node %s: %s", node.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "pv" || resType == "persistentvolume") && kind == "PersistentVolumeList" {
			var pvList corev1.PersistentVolumeList
			err := json.Unmarshal([]byte(buffer), &pvList)
			if err != nil {
				log.Fatalf("Error parsing pv list: %v\n%v\n", err.Error(), buffer)
			}
			for _, pv := range pvList.Items {
				if resName == pv.Name {
					s, err := describePersistentVolume(&pv)
					if err != nil {
						log.Fatalf("Error generating output for pv %s: %s", pv.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "po" || resType == "pod") && kind == "PodList" {
			var podList corev1.PodList
			err := json.Unmarshal([]byte(buffer), &podList)
			if err != nil {
				log.Fatalf("Error parsing pod list: %v\n%v\n", err.Error(), buffer)
			}
			for _, pod := range podList.Items {
				if resName == pod.Name && resNamespace == pod.Namespace {
					s, err := describePod(&pod)
					if err != nil {
						log.Fatalf("Error generating output for pod %s/%s: %s", pod.Namespace, pod.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "svc" || resType == "service") && kind == "ServiceList" {
			var svcList corev1.ServiceList
			err := json.Unmarshal([]byte(buffer), &svcList)
			if err != nil {
				log.Fatalf("Error parsing service list: %v\n%v\n", err.Error(), buffer)
			}
			for _, svc := range svcList.Items {
				if resName == svc.Name && resNamespace == svc.Namespace {
					s, err := describeService(&svc)
					if err != nil {
						log.Fatalf("Error generating output for service %s/%s: %s", svc.Namespace, svc.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "deploy" || resType == "deployment") && kind == "DeploymentList" {
			var deployList appsv1.DeploymentList
			err := json.Unmarshal([]byte(buffer), &deployList)
			if err != nil {
				log.Fatalf("Error parsing deployment list: %v\n%v\n", err.Error(), buffer)
			}
			for _, deploy := range deployList.Items {
				if resName == deploy.Name && resNamespace == deploy.Namespace {
					s, err := describeDeployment(&deploy)
					if err != nil {
						log.Fatalf("Error generating output for deployment %s/%s: %s", deploy.Namespace, deploy.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "ds" || resType == "daemonset") && kind == "DaemonSetList" {
			var daemonList appsv1.DaemonSetList
			err := json.Unmarshal([]byte(buffer), &daemonList)
			if err != nil {
				log.Fatalf("Error parsing daemonset list: %v\n%v\n", err.Error(), buffer)
			}
			for _, daemon := range daemonList.Items {
				if resName == daemon.Name && resNamespace == daemon.Namespace {
					s, err := describeDaemonSet(&daemon)
					if err != nil {
						log.Fatalf("Error generating output for daemonset %s/%s: %s", daemon.Namespace, daemon.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "rs" || resType == "replicaset") && kind == "ReplicaSetList" {
			var replicaList appsv1.ReplicaSetList
			err := json.Unmarshal([]byte(buffer), &replicaList)
			if err != nil {
				log.Fatalf("Error parsing replicaset list: %v\n%v\n", err.Error(), buffer)
			}
			for _, rs := range replicaList.Items {
				if resName == rs.Name && resNamespace == rs.Namespace {
					s, err := describeReplicaSet(&rs)
					if err != nil {
						log.Fatalf("Error generating output for replicaset %s/%s: %s", rs.Namespace, rs.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "sts" || resType == "statefulset") && kind == "StatefulSetList" {
			var stsList appsv1.StatefulSetList
			err := json.Unmarshal([]byte(buffer), &stsList)
			if err != nil {
				log.Fatalf("Error parsing statefulset list: %v\n%v\n", err.Error(), buffer)
			}
			for _, sts := range stsList.Items {
				if resName == sts.Name && resNamespace == sts.Namespace {
					s, err := describeStatefulSet(&sts)
					if err != nil {
						log.Fatalf("Error generating output for statefulset %s/%s: %s", sts.Namespace, sts.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "pvc" || resType == "persistentvolumeclaim") && kind == "PersistentVolumeClaimList" {
			var pvcList corev1.PersistentVolumeClaimList
			err := json.Unmarshal([]byte(buffer), &pvcList)
			if err != nil {
				log.Fatalf("Error parsing pvc list: %v\n%v\n", err.Error(), buffer)
			}
			for _, pvc := range pvcList.Items {
				if resName == pvc.Name && resNamespace == pvc.Namespace {
					s, err := describePersistentVolumeClaim(&pvc)
					if err != nil {
						log.Fatalf("Error generating output for pvc %s/%s: %s", pvc.Namespace, pvc.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "cm" || resType == "configmap") && kind == "ConfigMapList" {
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
		} else if (resType == "secret") && kind == "SecretList" {
			var secretList corev1.SecretList
			err := json.Unmarshal([]byte(buffer), &secretList)
			if err != nil {
				log.Fatalf("Error parsing secret list: %v\n%v\n", err.Error(), buffer)
			}
			for _, scrt := range secretList.Items {
				if resName == scrt.Name && resNamespace == scrt.Namespace {
					s, err := describeSecret(&scrt)
					if err != nil {
						log.Fatalf("Error generating output for secret %s/%s: %s", scrt.Namespace, scrt.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "sa" || resType == "serviceaccount") && kind == "ServiceAccountList" {
			var saList corev1.ServiceAccountList
			err := json.Unmarshal([]byte(buffer), &saList)
			if err != nil {
				log.Fatalf("Error parsing sa list: %v\n%v\n", err.Error(), buffer)
			}
			for _, sa := range saList.Items {
				if resName == sa.Name && resNamespace == sa.Namespace {
					s, err := describeServiceAccount(&sa)
					if err != nil {
						log.Fatalf("Error generating output for sa %s/%s: %s", sa.Namespace, sa.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		} else if (resType == "ing" || resType == "ingress") && kind == "IngressList" {
			var ingList networkingv1.IngressList
			err := json.Unmarshal([]byte(buffer), &ingList)
			if err != nil {
				log.Fatalf("Error parsing ingress list: %v\n%v\n", err.Error(), buffer)
			}
			for _, ing := range ingList.Items {
				if resName == ing.Name && resNamespace == ing.Namespace {
					s, err := describeIngress(&ing)
					if err != nil {
						log.Fatalf("Error generating output for ingress %s/%s: %s", ing.Namespace, ing.Name, err.Error())
					}
					fmt.Println(s)
					break
				}
			}
		}

	}
}

func describeIngress(ing *networkingv1.Ingress) (string, error) {
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

func describeServiceAccount(serviceAccount *corev1.ServiceAccount) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", serviceAccount.Name)
		w.Write(LEVEL_0, "Namespace:\t%s\n", serviceAccount.Namespace)
		printLabelsMultiline(w, "Labels", serviceAccount.Labels)
		printAnnotationsMultiline(w, "Annotations", serviceAccount.Annotations)

		var (
			emptyHeader = "                   "
			pullHeader  = "Image pull secrets:"
			mountHeader = "Mountable secrets: "
			tokenHeader = "Tokens:            "

			pullSecretNames  = []string{}
			mountSecretNames = []string{}
			tokenSecretNames = []string{}
		)

		for _, s := range serviceAccount.ImagePullSecrets {
			pullSecretNames = append(pullSecretNames, s.Name)
		}
		for _, s := range serviceAccount.Secrets {
			mountSecretNames = append(mountSecretNames, s.Name)
		}
		// for _, s := range tokens {
		// 	tokenSecretNames = append(tokenSecretNames, s.Name)
		// }

		types := map[string][]string{
			pullHeader:  pullSecretNames,
			mountHeader: mountSecretNames,
			tokenHeader: tokenSecretNames,
		}
		for _, header := range sets.StringKeySet(types).List() {
			names := types[header]
			if len(names) == 0 {
				w.Write(LEVEL_0, "%s\t<none>\n", header)
			} else {
				prefix := header
				for _, name := range names {
					// if missingSecrets.Has(name) {
					// 	w.Write(LEVEL_0, "%s\t%s (not found)\n", prefix, name)
					// } else {
					w.Write(LEVEL_0, "%s\t%s\n", prefix, name)
					// }
					prefix = emptyHeader
				}
			}
		}

		return nil
	})
}

func describeSecret(secret *corev1.Secret) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", secret.Name)
		w.Write(LEVEL_0, "Namespace:\t%s\n", secret.Namespace)
		printLabelsMultiline(w, "Labels", secret.Labels)
		printAnnotationsMultiline(w, "Annotations", secret.Annotations)

		w.Write(LEVEL_0, "\nType:\t%s\n", secret.Type)

		w.Write(LEVEL_0, "\nData\n====\n")
		for k, v := range secret.Data {
			switch {
			case k == corev1.ServiceAccountTokenKey && secret.Type == corev1.SecretTypeServiceAccountToken:
				w.Write(LEVEL_0, "%s:\t%s\n", k, string(v))
			default:
				w.Write(LEVEL_0, "%s:\t%d bytes\n", k, len(v))
			}
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

func describeStatefulSet(ps *appsv1.StatefulSet) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", ps.ObjectMeta.Name)
		w.Write(LEVEL_0, "Namespace:\t%s\n", ps.ObjectMeta.Namespace)
		w.Write(LEVEL_0, "CreationTimestamp:\t%s\n", ps.CreationTimestamp.Time.Format(time.RFC1123Z))
		selector, err := metav1.LabelSelectorAsSelector(ps.Spec.Selector)
		if err != nil {
			return err
		}
		w.Write(LEVEL_0, "Selector:\t%s\n", selector)
		printLabelsMultiline(w, "Labels", ps.Labels)
		printAnnotationsMultiline(w, "Annotations", ps.Annotations)
		w.Write(LEVEL_0, "Replicas:\t%d desired | %d total\n", *ps.Spec.Replicas, ps.Status.Replicas)
		w.Write(LEVEL_0, "Update Strategy:\t%s\n", ps.Spec.UpdateStrategy.Type)
		if ps.Spec.UpdateStrategy.RollingUpdate != nil {
			ru := ps.Spec.UpdateStrategy.RollingUpdate
			if ru.Partition != nil {
				w.Write(LEVEL_1, "Partition:\t%d\n", *ru.Partition)
				if ru.MaxUnavailable != nil {
					w.Write(LEVEL_1, "MaxUnavailable:\t%s\n", ru.MaxUnavailable.String())
				}
			}
		}

		// w.Write(LEVEL_0, "Pods Status:\t%d Running / %d Waiting / %d Succeeded / %d Failed\n", running, waiting, succeeded, failed)
		DescribePodTemplate(&ps.Spec.Template, w)
		describeVolumeClaimTemplates(ps.Spec.VolumeClaimTemplates, w)

		return nil
	})
}

func describeVolumeClaimTemplates(templates []corev1.PersistentVolumeClaim, w PrefixWriter) {
	if len(templates) == 0 {
		w.Write(LEVEL_0, "Volume Claims:\t<none>\n")
		return
	}
	w.Write(LEVEL_0, "Volume Claims:\n")
	for _, pvc := range templates {
		w.Write(LEVEL_1, "Name:\t%s\n", pvc.Name)
		w.Write(LEVEL_1, "StorageClass:\t%s\n", storageutil.GetPersistentVolumeClaimClass(&pvc))
		printLabelsMultilineWithIndent(w, "  ", "Labels", "\t", pvc.Labels, sets.NewString())
		printLabelsMultilineWithIndent(w, "  ", "Annotations", "\t", pvc.Annotations, sets.NewString())
		if capacity, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
			w.Write(LEVEL_1, "Capacity:\t%s\n", capacity.String())
		} else {
			w.Write(LEVEL_1, "Capacity:\t%s\n", "<default>")
		}
		w.Write(LEVEL_1, "Access Modes:\t%s\n", pvc.Spec.AccessModes)
	}
}

func describePersistentVolume(pv *corev1.PersistentVolume) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", pv.Name)
		printLabelsMultiline(w, "Labels", pv.ObjectMeta.Labels)
		printAnnotationsMultiline(w, "Annotations", pv.ObjectMeta.Annotations)
		w.Write(LEVEL_0, "Finalizers:\t%v\n", pv.ObjectMeta.Finalizers)
		w.Write(LEVEL_0, "StorageClass:\t%s\n", storageutil.GetPersistentVolumeClass(pv))
		if pv.ObjectMeta.DeletionTimestamp != nil {
			w.Write(LEVEL_0, "Status:\tTerminating (lasts %s)\n", translateTimestampSince(*pv.ObjectMeta.DeletionTimestamp))
		} else {
			w.Write(LEVEL_0, "Status:\t%v\n", pv.Status.Phase)
		}
		if pv.Spec.ClaimRef != nil {
			w.Write(LEVEL_0, "Claim:\t%s\n", pv.Spec.ClaimRef.Namespace+"/"+pv.Spec.ClaimRef.Name)
		} else {
			w.Write(LEVEL_0, "Claim:\t%s\n", "")
		}
		w.Write(LEVEL_0, "Reclaim Policy:\t%v\n", pv.Spec.PersistentVolumeReclaimPolicy)
		w.Write(LEVEL_0, "Access Modes:\t%s\n", storageutil.GetAccessModesAsString(pv.Spec.AccessModes))
		if pv.Spec.VolumeMode != nil {
			w.Write(LEVEL_0, "VolumeMode:\t%v\n", *pv.Spec.VolumeMode)
		}
		storage := pv.Spec.Capacity[corev1.ResourceStorage]
		w.Write(LEVEL_0, "Capacity:\t%s\n", storage.String())
		printVolumeNodeAffinity(w, pv.Spec.NodeAffinity)
		w.Write(LEVEL_0, "Message:\t%s\n", pv.Status.Message)
		w.Write(LEVEL_0, "Source:\n")

		switch {
		case pv.Spec.HostPath != nil:
			printHostPathVolumeSource(pv.Spec.HostPath, w)
		case pv.Spec.GCEPersistentDisk != nil:
			printGCEPersistentDiskVolumeSource(pv.Spec.GCEPersistentDisk, w)
		case pv.Spec.AWSElasticBlockStore != nil:
			printAWSElasticBlockStoreVolumeSource(pv.Spec.AWSElasticBlockStore, w)
		case pv.Spec.NFS != nil:
			printNFSVolumeSource(pv.Spec.NFS, w)
		case pv.Spec.ISCSI != nil:
			printISCSIPersistentVolumeSource(pv.Spec.ISCSI, w)
		case pv.Spec.Glusterfs != nil:
			printGlusterfsPersistentVolumeSource(pv.Spec.Glusterfs, w)
		case pv.Spec.RBD != nil:
			printRBDPersistentVolumeSource(pv.Spec.RBD, w)
		case pv.Spec.Quobyte != nil:
			printQuobyteVolumeSource(pv.Spec.Quobyte, w)
		case pv.Spec.VsphereVolume != nil:
			printVsphereVolumeSource(pv.Spec.VsphereVolume, w)
		case pv.Spec.Cinder != nil:
			printCinderPersistentVolumeSource(pv.Spec.Cinder, w)
		case pv.Spec.AzureDisk != nil:
			printAzureDiskVolumeSource(pv.Spec.AzureDisk, w)
		case pv.Spec.PhotonPersistentDisk != nil:
			printPhotonPersistentDiskVolumeSource(pv.Spec.PhotonPersistentDisk, w)
		case pv.Spec.PortworxVolume != nil:
			printPortworxVolumeSource(pv.Spec.PortworxVolume, w)
		case pv.Spec.ScaleIO != nil:
			printScaleIOPersistentVolumeSource(pv.Spec.ScaleIO, w)
		case pv.Spec.Local != nil:
			printLocalVolumeSource(pv.Spec.Local, w)
		case pv.Spec.CephFS != nil:
			printCephFSPersistentVolumeSource(pv.Spec.CephFS, w)
		case pv.Spec.StorageOS != nil:
			printStorageOSPersistentVolumeSource(pv.Spec.StorageOS, w)
		case pv.Spec.FC != nil:
			printFCVolumeSource(pv.Spec.FC, w)
		case pv.Spec.AzureFile != nil:
			printAzureFilePersistentVolumeSource(pv.Spec.AzureFile, w)
		case pv.Spec.FlexVolume != nil:
			printFlexPersistentVolumeSource(pv.Spec.FlexVolume, w)
		case pv.Spec.Flocker != nil:
			printFlockerVolumeSource(pv.Spec.Flocker, w)
		case pv.Spec.CSI != nil:
			printCSIPersistentVolumeSource(pv.Spec.CSI, w)
		default:
			w.Write(LEVEL_1, "<unknown>\n")
		}

		return nil
	})
}

func printVolumeNodeAffinity(w PrefixWriter, affinity *corev1.VolumeNodeAffinity) {
	w.Write(LEVEL_0, "Node Affinity:\t")
	if affinity == nil || affinity.Required == nil {
		w.WriteLine("<none>")
		return
	}
	w.WriteLine("")

	if affinity.Required != nil {
		w.Write(LEVEL_1, "Required Terms:\t")
		if len(affinity.Required.NodeSelectorTerms) == 0 {
			w.WriteLine("<none>")
		} else {
			w.WriteLine("")
			for i, term := range affinity.Required.NodeSelectorTerms {
				printNodeSelectorTermsMultilineWithIndent(w, LEVEL_2, fmt.Sprintf("Term %v", i), "\t", term.MatchExpressions)
			}
		}
	}
}

// printLabelsMultiline prints multiple labels with a user-defined alignment.
func printNodeSelectorTermsMultilineWithIndent(w PrefixWriter, indentLevel int, title, innerIndent string, reqs []corev1.NodeSelectorRequirement) {
	w.Write(indentLevel, "%s:%s", title, innerIndent)

	if len(reqs) == 0 {
		w.WriteLine("<none>")
		return
	}

	for i, req := range reqs {
		if i != 0 {
			w.Write(indentLevel, "%s", innerIndent)
		}
		exprStr := fmt.Sprintf("%s %s", req.Key, strings.ToLower(string(req.Operator)))
		if len(req.Values) > 0 {
			exprStr = fmt.Sprintf("%s [%s]", exprStr, strings.Join(req.Values, ", "))
		}
		w.Write(LEVEL_0, "%s\n", exprStr)
	}
}

func describePersistentVolumeClaim(pvc *corev1.PersistentVolumeClaim) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		printPersistentVolumeClaim(w, pvc, true)
		// printPodsMultiline(w, "Used By", pods)

		if len(pvc.Status.Conditions) > 0 {
			w.Write(LEVEL_0, "Conditions:\n")
			w.Write(LEVEL_1, "Type\tStatus\tLastProbeTime\tLastTransitionTime\tReason\tMessage\n")
			w.Write(LEVEL_1, "----\t------\t-----------------\t------------------\t------\t-------\n")
			for _, c := range pvc.Status.Conditions {
				w.Write(LEVEL_1, "%v \t%v \t%s \t%s \t%v \t%v\n",
					c.Type,
					c.Status,
					c.LastProbeTime.Time.Format(time.RFC1123Z),
					c.LastTransitionTime.Time.Format(time.RFC1123Z),
					c.Reason,
					c.Message)
			}
		}

		return nil
	})
}

func describeNode(node corev1.Node) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", node.Name)
		if roles := findNodeRoles(node); len(roles) > 0 {
			w.Write(LEVEL_0, "Roles:\t%s\n", strings.Join(roles, ","))
		} else {
			w.Write(LEVEL_0, "Roles:\t%s\n", "<none>")
		}
		printLabelsMultiline(w, "Labels", node.Labels)
		printAnnotationsMultiline(w, "Annotations", node.Annotations)
		w.Write(LEVEL_0, "CreationTimestamp:\t%s\n", node.CreationTimestamp.Time.Format(time.RFC1123Z))
		printNodeTaintsMultiline(w, "Taints", node.Spec.Taints)
		w.Write(LEVEL_0, "Unschedulable:\t%v\n", node.Spec.Unschedulable)

		if len(node.Status.Conditions) > 0 {
			w.Write(LEVEL_0, "Conditions:\n  Type\tStatus\tLastHeartbeatTime\tLastTransitionTime\tReason\tMessage\n")
			w.Write(LEVEL_1, "----\t------\t-----------------\t------------------\t------\t-------\n")
			for _, c := range node.Status.Conditions {
				w.Write(LEVEL_1, "%v \t%v \t%s \t%s \t%v \t%v\n",
					c.Type,
					c.Status,
					c.LastHeartbeatTime.Time.Format(time.RFC1123Z),
					c.LastTransitionTime.Time.Format(time.RFC1123Z),
					c.Reason,
					c.Message)
			}
		}

		w.Write(LEVEL_0, "Addresses:\n")
		for _, address := range node.Status.Addresses {
			w.Write(LEVEL_1, "%s:\t%s\n", address.Type, address.Address)
		}

		printResourceList := func(resourceList corev1.ResourceList) {
			resources := make([]corev1.ResourceName, 0, len(resourceList))
			for resource := range resourceList {
				resources = append(resources, resource)
			}
			sort.Sort(SortableResourceNames(resources))
			for _, resource := range resources {
				value := resourceList[resource]
				w.Write(LEVEL_0, "  %s:\t%s\n", resource, value.String())
			}
		}

		if len(node.Status.Capacity) > 0 {
			w.Write(LEVEL_0, "Capacity:\n")
			printResourceList(node.Status.Capacity)
		}
		if len(node.Status.Allocatable) > 0 {
			w.Write(LEVEL_0, "Allocatable:\n")
			printResourceList(node.Status.Allocatable)
		}

		w.Write(LEVEL_0, "System Info:\n")
		w.Write(LEVEL_0, "  Machine ID:\t%s\n", node.Status.NodeInfo.MachineID)
		w.Write(LEVEL_0, "  System UUID:\t%s\n", node.Status.NodeInfo.SystemUUID)
		w.Write(LEVEL_0, "  Boot ID:\t%s\n", node.Status.NodeInfo.BootID)
		w.Write(LEVEL_0, "  Kernel Version:\t%s\n", node.Status.NodeInfo.KernelVersion)
		w.Write(LEVEL_0, "  OS Image:\t%s\n", node.Status.NodeInfo.OSImage)
		w.Write(LEVEL_0, "  Operating System:\t%s\n", node.Status.NodeInfo.OperatingSystem)
		w.Write(LEVEL_0, "  Architecture:\t%s\n", node.Status.NodeInfo.Architecture)
		w.Write(LEVEL_0, "  Container Runtime Version:\t%s\n", node.Status.NodeInfo.ContainerRuntimeVersion)
		w.Write(LEVEL_0, "  Kubelet Version:\t%s\n", node.Status.NodeInfo.KubeletVersion)
		w.Write(LEVEL_0, "  Kube-Proxy Version:\t%s\n", node.Status.NodeInfo.KubeProxyVersion)

		// remove when .PodCIDR is depreciated
		if len(node.Spec.PodCIDR) > 0 {
			w.Write(LEVEL_0, "PodCIDR:\t%s\n", node.Spec.PodCIDR)
		}

		if len(node.Spec.PodCIDRs) > 0 {
			w.Write(LEVEL_0, "PodCIDRs:\t%s\n", strings.Join(node.Spec.PodCIDRs, ","))
		}
		if len(node.Spec.ProviderID) > 0 {
			w.Write(LEVEL_0, "ProviderID:\t%s\n", node.Spec.ProviderID)
		}

		return nil
	})
}

// findNodeRoles returns the roles of a given node.
// The roles are determined by looking for:
// * a node-role.kubernetes.io/<role>="" label
// * a kubernetes.io/role="<role>" label
func findNodeRoles(node corev1.Node) []string {
	roles := sets.NewString()
	for k, v := range node.Labels {
		switch {
		case strings.HasPrefix(k, LabelNodeRolePrefix):
			if role := strings.TrimPrefix(k, LabelNodeRolePrefix); len(role) > 0 {
				roles.Insert(role)
			}

		case k == NodeLabelRole && v != "":
			roles.Insert(v)
		}
	}
	return roles.List()
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

// printTaintsMultiline prints multiple taints with a proper alignment.
func printNodeTaintsMultiline(w PrefixWriter, title string, taints []corev1.Taint) {
	printTaintsMultilineWithIndent(w, "", title, "\t", taints)
}

// printTaintsMultilineWithIndent prints multiple taints with a user-defined alignment.
func printTaintsMultilineWithIndent(w PrefixWriter, initialIndent, title, innerIndent string, taints []corev1.Taint) {
	w.Write(LEVEL_0, "%s%s:%s", initialIndent, title, innerIndent)

	if len(taints) == 0 {
		w.WriteLine("<none>")
		return
	}

	// to print taints in the sorted order
	sort.Slice(taints, func(i, j int) bool {
		cmpKey := func(taint corev1.Taint) string {
			return string(taint.Effect) + "," + taint.Key
		}
		return cmpKey(taints[i]) < cmpKey(taints[j])
	})

	for i, taint := range taints {
		if i != 0 {
			w.Write(LEVEL_0, "%s", initialIndent)
			w.Write(LEVEL_0, "%s", innerIndent)
		}
		w.Write(LEVEL_0, "%s\n", taint.ToString())
	}
}

func describePod(pod *corev1.Pod) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", pod.Name)
		w.Write(LEVEL_0, "Namespace:\t%s\n", pod.Namespace)
		if pod.Spec.Priority != nil {
			w.Write(LEVEL_0, "Priority:\t%d\n", *pod.Spec.Priority)
		}
		if len(pod.Spec.PriorityClassName) > 0 {
			w.Write(LEVEL_0, "Priority Class Name:\t%s\n", pod.Spec.PriorityClassName)
		}
		if pod.Spec.RuntimeClassName != nil && len(*pod.Spec.RuntimeClassName) > 0 {
			w.Write(LEVEL_0, "Runtime Class Name:\t%s\n", *pod.Spec.RuntimeClassName)
		}
		if len(pod.Spec.ServiceAccountName) > 0 {
			w.Write(LEVEL_0, "Service Account:\t%s\n", pod.Spec.ServiceAccountName)
		}
		if pod.Spec.NodeName == "" {
			w.Write(LEVEL_0, "Node:\t<none>\n")
		} else {
			w.Write(LEVEL_0, "Node:\t%s\n", pod.Spec.NodeName+"/"+pod.Status.HostIP)
		}
		if pod.Status.StartTime != nil {
			w.Write(LEVEL_0, "Start Time:\t%s\n", pod.Status.StartTime.Time.Format(time.RFC1123Z))
		}
		printLabelsMultiline(w, "Labels", pod.Labels)
		printAnnotationsMultiline(w, "Annotations", pod.Annotations)
		if pod.DeletionTimestamp != nil {
			w.Write(LEVEL_0, "Status:\tTerminating (lasts %s)\n", translateTimestampSince(*pod.DeletionTimestamp))
			w.Write(LEVEL_0, "Termination Grace Period:\t%ds\n", *pod.DeletionGracePeriodSeconds)
		} else {
			w.Write(LEVEL_0, "Status:\t%s\n", string(pod.Status.Phase))
		}
		if len(pod.Status.Reason) > 0 {
			w.Write(LEVEL_0, "Reason:\t%s\n", pod.Status.Reason)
		}
		if len(pod.Status.Message) > 0 {
			w.Write(LEVEL_0, "Message:\t%s\n", pod.Status.Message)
		}
		if pod.Spec.SecurityContext != nil && pod.Spec.SecurityContext.SeccompProfile != nil {
			w.Write(LEVEL_0, "SeccompProfile:\t%s\n", pod.Spec.SecurityContext.SeccompProfile.Type)
			if pod.Spec.SecurityContext.SeccompProfile.Type == corev1.SeccompProfileTypeLocalhost {
				w.Write(LEVEL_0, "LocalhostProfile:\t%s\n", *pod.Spec.SecurityContext.SeccompProfile.LocalhostProfile)
			}
		}
		// remove when .IP field is depreciated
		w.Write(LEVEL_0, "IP:\t%s\n", pod.Status.PodIP)
		describePodIPs(pod, w, "")
		if controlledBy := printController(pod); len(controlledBy) > 0 {
			w.Write(LEVEL_0, "Controlled By:\t%s\n", controlledBy)
		}
		if len(pod.Status.NominatedNodeName) > 0 {
			w.Write(LEVEL_0, "NominatedNodeName:\t%s\n", pod.Status.NominatedNodeName)
		}

		if len(pod.Spec.InitContainers) > 0 {
			describeContainers("Init Containers", pod.Spec.InitContainers, pod.Status.InitContainerStatuses, EnvValueRetriever(pod), w, "")
		}
		describeContainers("Containers", pod.Spec.Containers, pod.Status.ContainerStatuses, EnvValueRetriever(pod), w, "")
		if len(pod.Spec.EphemeralContainers) > 0 {
			var ec []corev1.Container
			for i := range pod.Spec.EphemeralContainers {
				ec = append(ec, corev1.Container(pod.Spec.EphemeralContainers[i].EphemeralContainerCommon))
			}
			describeContainers("Ephemeral Containers", ec, pod.Status.EphemeralContainerStatuses, EnvValueRetriever(pod), w, "")
		}
		if len(pod.Spec.ReadinessGates) > 0 {
			w.Write(LEVEL_0, "Readiness Gates:\n  Type\tStatus\n")
			for _, g := range pod.Spec.ReadinessGates {
				status := "<none>"
				for _, c := range pod.Status.Conditions {
					if c.Type == g.ConditionType {
						status = fmt.Sprintf("%v", c.Status)
						break
					}
				}
				w.Write(LEVEL_1, "%v \t%v \n",
					g.ConditionType,
					status)
			}
		}
		if len(pod.Status.Conditions) > 0 {
			w.Write(LEVEL_0, "Conditions:\n  Type\tStatus\n")
			for _, c := range pod.Status.Conditions {
				w.Write(LEVEL_1, "%v \t%v \n",
					c.Type,
					c.Status)
			}
		}
		describeVolumes(pod.Spec.Volumes, w, "")
		if pod.Status.QOSClass != "" {
			w.Write(LEVEL_0, "QoS Class:\t%s\n", pod.Status.QOSClass)
		} else {
			w.Write(LEVEL_0, "QoS Class:\t%s\n", qos.GetPodQOS(pod))
		}
		printLabelsMultiline(w, "Node-Selectors", pod.Spec.NodeSelector)
		printPodTolerationsMultiline(w, "Tolerations", pod.Spec.Tolerations)
		describeTopologySpreadConstraints(pod.Spec.TopologySpreadConstraints, w, "")
		return nil
	})
}

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

func describePodIPs(pod *corev1.Pod, w PrefixWriter, space string) {
	if len(pod.Status.PodIPs) == 0 {
		w.Write(LEVEL_0, "%sIPs:\t<none>\n", space)
		return
	}
	w.Write(LEVEL_0, "%sIPs:\n", space)
	for _, ipInfo := range pod.Status.PodIPs {
		w.Write(LEVEL_1, "IP:\t%s\n", ipInfo.IP)
	}
}

func printController(controllee metav1.Object) string {
	if controllerRef := metav1.GetControllerOf(controllee); controllerRef != nil {
		return fmt.Sprintf("%s/%s", controllerRef.Kind, controllerRef.Name)
	}
	return ""
}

func describeContainers(label string, containers []corev1.Container, containerStatuses []corev1.ContainerStatus,
	resolverFn EnvVarResolverFunc, w PrefixWriter, space string) {
	statuses := map[string]corev1.ContainerStatus{}
	for _, status := range containerStatuses {
		statuses[status.Name] = status
	}

	describeContainersLabel(containers, label, space, w)

	for _, container := range containers {
		status, ok := statuses[container.Name]
		describeContainerBasicInfo(container, status, ok, space, w)
		describeContainerCommand(container, w)
		if ok {
			describeContainerState(status, w)
		}
		describeContainerResource(container, w)
		describeContainerProbe(container, w)
		if len(container.EnvFrom) > 0 {
			describeContainerEnvFrom(container, resolverFn, w)
		}
		describeContainerEnvVars(container, resolverFn, w)
		describeContainerVolumes(container, w)
	}
}

func describeContainersLabel(containers []corev1.Container, label, space string, w PrefixWriter) {
	none := ""
	if len(containers) == 0 {
		none = " <none>"
	}
	w.Write(LEVEL_0, "%s%s:%s\n", space, label, none)
}

func describeContainerBasicInfo(container corev1.Container, status corev1.ContainerStatus, ok bool, space string, w PrefixWriter) {
	nameIndent := ""
	if len(space) > 0 {
		nameIndent = " "
	}
	w.Write(LEVEL_1, "%s%v:\n", nameIndent, container.Name)
	if ok {
		w.Write(LEVEL_2, "Container ID:\t%s\n", status.ContainerID)
	}
	w.Write(LEVEL_2, "Image:\t%s\n", container.Image)
	if ok {
		w.Write(LEVEL_2, "Image ID:\t%s\n", status.ImageID)
	}
	portString := describeContainerPorts(container.Ports)
	if strings.Contains(portString, ",") {
		w.Write(LEVEL_2, "Ports:\t%s\n", portString)
	} else {
		w.Write(LEVEL_2, "Port:\t%s\n", stringOrNone(portString))
	}
	hostPortString := describeContainerHostPorts(container.Ports)
	if strings.Contains(hostPortString, ",") {
		w.Write(LEVEL_2, "Host Ports:\t%s\n", hostPortString)
	} else {
		w.Write(LEVEL_2, "Host Port:\t%s\n", stringOrNone(hostPortString))
	}
	if container.SecurityContext != nil && container.SecurityContext.SeccompProfile != nil {
		w.Write(LEVEL_2, "SeccompProfile:\t%s\n", container.SecurityContext.SeccompProfile.Type)
		if container.SecurityContext.SeccompProfile.Type == corev1.SeccompProfileTypeLocalhost {
			w.Write(LEVEL_3, "LocalhostProfile:\t%s\n", *container.SecurityContext.SeccompProfile.LocalhostProfile)
		}
	}
}

func describeContainerPorts(cPorts []corev1.ContainerPort) string {
	ports := make([]string, 0, len(cPorts))
	for _, cPort := range cPorts {
		ports = append(ports, fmt.Sprintf("%d/%s", cPort.ContainerPort, cPort.Protocol))
	}
	return strings.Join(ports, ", ")
}

func describeContainerHostPorts(cPorts []corev1.ContainerPort) string {
	ports := make([]string, 0, len(cPorts))
	for _, cPort := range cPorts {
		ports = append(ports, fmt.Sprintf("%d/%s", cPort.HostPort, cPort.Protocol))
	}
	return strings.Join(ports, ", ")
}

func describeContainerCommand(container corev1.Container, w PrefixWriter) {
	if len(container.Command) > 0 {
		w.Write(LEVEL_2, "Command:\n")
		for _, c := range container.Command {
			for _, s := range strings.Split(c, "\n") {
				w.Write(LEVEL_3, "%s\n", s)
			}
		}
	}
	if len(container.Args) > 0 {
		w.Write(LEVEL_2, "Args:\n")
		for _, arg := range container.Args {
			for _, s := range strings.Split(arg, "\n") {
				w.Write(LEVEL_3, "%s\n", s)
			}
		}
	}
}

func describeContainerResource(container corev1.Container, w PrefixWriter) {
	resources := container.Resources
	if len(resources.Limits) > 0 {
		w.Write(LEVEL_2, "Limits:\n")
	}
	for _, name := range SortedResourceNames(resources.Limits) {
		quantity := resources.Limits[name]
		w.Write(LEVEL_3, "%s:\t%s\n", name, quantity.String())
	}

	if len(resources.Requests) > 0 {
		w.Write(LEVEL_2, "Requests:\n")
	}
	for _, name := range SortedResourceNames(resources.Requests) {
		quantity := resources.Requests[name]
		w.Write(LEVEL_3, "%s:\t%s\n", name, quantity.String())
	}
}

func describeContainerState(status corev1.ContainerStatus, w PrefixWriter) {
	describeStatus("State", status.State, w)
	if status.LastTerminationState.Terminated != nil {
		describeStatus("Last State", status.LastTerminationState, w)
	}
	w.Write(LEVEL_2, "Ready:\t%v\n", printBool(status.Ready))
	w.Write(LEVEL_2, "Restart Count:\t%d\n", status.RestartCount)
}

func describeContainerProbe(container corev1.Container, w PrefixWriter) {
	if container.LivenessProbe != nil {
		probe := DescribeProbe(container.LivenessProbe)
		w.Write(LEVEL_2, "Liveness:\t%s\n", probe)
	}
	if container.ReadinessProbe != nil {
		probe := DescribeProbe(container.ReadinessProbe)
		w.Write(LEVEL_2, "Readiness:\t%s\n", probe)
	}
	if container.StartupProbe != nil {
		probe := DescribeProbe(container.StartupProbe)
		w.Write(LEVEL_2, "Startup:\t%s\n", probe)
	}
}

func describeContainerVolumes(container corev1.Container, w PrefixWriter) {
	// Show volumeMounts
	none := ""
	if len(container.VolumeMounts) == 0 {
		none = "\t<none>"
	}
	w.Write(LEVEL_2, "Mounts:%s\n", none)
	sort.Sort(SortableVolumeMounts(container.VolumeMounts))
	for _, mount := range container.VolumeMounts {
		flags := []string{}
		if mount.ReadOnly {
			flags = append(flags, "ro")
		} else {
			flags = append(flags, "rw")
		}
		if len(mount.SubPath) > 0 {
			flags = append(flags, fmt.Sprintf("path=%q", mount.SubPath))
		}
		w.Write(LEVEL_3, "%s from %s (%s)\n", mount.MountPath, mount.Name, strings.Join(flags, ","))
	}
	// Show volumeDevices if exists
	if len(container.VolumeDevices) > 0 {
		w.Write(LEVEL_2, "Devices:%s\n", none)
		sort.Sort(SortableVolumeDevices(container.VolumeDevices))
		for _, device := range container.VolumeDevices {
			w.Write(LEVEL_3, "%s from %s\n", device.DevicePath, device.Name)
		}
	}
}

func describeContainerEnvVars(container corev1.Container, resolverFn EnvVarResolverFunc, w PrefixWriter) {
	none := ""
	if len(container.Env) == 0 {
		none = "\t<none>"
	}
	w.Write(LEVEL_2, "Environment:%s\n", none)

	for _, e := range container.Env {
		if e.ValueFrom == nil {
			for i, s := range strings.Split(e.Value, "\n") {
				if i == 0 {
					w.Write(LEVEL_3, "%s:\t%s\n", e.Name, s)
				} else {
					w.Write(LEVEL_3, "\t%s\n", s)
				}
			}
			continue
		}

		switch {
		case e.ValueFrom.FieldRef != nil:
			var valueFrom string
			if resolverFn != nil {
				valueFrom = resolverFn(e)
			}
			w.Write(LEVEL_3, "%s:\t%s (%s:%s)\n", e.Name, valueFrom, e.ValueFrom.FieldRef.APIVersion, e.ValueFrom.FieldRef.FieldPath)
		case e.ValueFrom.ResourceFieldRef != nil:
			valueFrom, err := resourcehelper.ExtractContainerResourceValue(e.ValueFrom.ResourceFieldRef, &container)
			if err != nil {
				valueFrom = ""
			}
			resource := e.ValueFrom.ResourceFieldRef.Resource
			if valueFrom == "0" && (resource == "limits.cpu" || resource == "limits.memory") {
				valueFrom = "node allocatable"
			}
			w.Write(LEVEL_3, "%s:\t%s (%s)\n", e.Name, valueFrom, resource)
		case e.ValueFrom.SecretKeyRef != nil:
			optional := e.ValueFrom.SecretKeyRef.Optional != nil && *e.ValueFrom.SecretKeyRef.Optional
			w.Write(LEVEL_3, "%s:\t<set to the key '%s' in secret '%s'>\tOptional: %t\n", e.Name, e.ValueFrom.SecretKeyRef.Key, e.ValueFrom.SecretKeyRef.Name, optional)
		case e.ValueFrom.ConfigMapKeyRef != nil:
			optional := e.ValueFrom.ConfigMapKeyRef.Optional != nil && *e.ValueFrom.ConfigMapKeyRef.Optional
			w.Write(LEVEL_3, "%s:\t<set to the key '%s' of config map '%s'>\tOptional: %t\n", e.Name, e.ValueFrom.ConfigMapKeyRef.Key, e.ValueFrom.ConfigMapKeyRef.Name, optional)
		}
	}
}

func describeContainerEnvFrom(container corev1.Container, resolverFn EnvVarResolverFunc, w PrefixWriter) {
	none := ""
	if len(container.EnvFrom) == 0 {
		none = "\t<none>"
	}
	w.Write(LEVEL_2, "Environment Variables from:%s\n", none)

	for _, e := range container.EnvFrom {
		from := ""
		name := ""
		optional := false
		if e.ConfigMapRef != nil {
			from = "ConfigMap"
			name = e.ConfigMapRef.Name
			optional = e.ConfigMapRef.Optional != nil && *e.ConfigMapRef.Optional
		} else if e.SecretRef != nil {
			from = "Secret"
			name = e.SecretRef.Name
			optional = e.SecretRef.Optional != nil && *e.SecretRef.Optional
		}
		if len(e.Prefix) == 0 {
			w.Write(LEVEL_3, "%s\t%s\tOptional: %t\n", name, from, optional)
		} else {
			w.Write(LEVEL_3, "%s\t%s with prefix '%s'\tOptional: %t\n", name, from, e.Prefix, optional)
		}
	}
}

func describeVolumes(volumes []corev1.Volume, w PrefixWriter, space string) {
	if len(volumes) == 0 {
		w.Write(LEVEL_0, "%sVolumes:\t<none>\n", space)
		return
	}

	w.Write(LEVEL_0, "%sVolumes:\n", space)
	for _, volume := range volumes {
		nameIndent := ""
		if len(space) > 0 {
			nameIndent = " "
		}
		w.Write(LEVEL_1, "%s%v:\n", nameIndent, volume.Name)
		switch {
		case volume.VolumeSource.HostPath != nil:
			printHostPathVolumeSource(volume.VolumeSource.HostPath, w)
		case volume.VolumeSource.EmptyDir != nil:
			printEmptyDirVolumeSource(volume.VolumeSource.EmptyDir, w)
		case volume.VolumeSource.GCEPersistentDisk != nil:
			printGCEPersistentDiskVolumeSource(volume.VolumeSource.GCEPersistentDisk, w)
		case volume.VolumeSource.AWSElasticBlockStore != nil:
			printAWSElasticBlockStoreVolumeSource(volume.VolumeSource.AWSElasticBlockStore, w)
		case volume.VolumeSource.GitRepo != nil:
			printGitRepoVolumeSource(volume.VolumeSource.GitRepo, w)
		case volume.VolumeSource.Secret != nil:
			printSecretVolumeSource(volume.VolumeSource.Secret, w)
		case volume.VolumeSource.ConfigMap != nil:
			printConfigMapVolumeSource(volume.VolumeSource.ConfigMap, w)
		case volume.VolumeSource.NFS != nil:
			printNFSVolumeSource(volume.VolumeSource.NFS, w)
		case volume.VolumeSource.ISCSI != nil:
			printISCSIVolumeSource(volume.VolumeSource.ISCSI, w)
		case volume.VolumeSource.Glusterfs != nil:
			printGlusterfsVolumeSource(volume.VolumeSource.Glusterfs, w)
		case volume.VolumeSource.PersistentVolumeClaim != nil:
			printPersistentVolumeClaimVolumeSource(volume.VolumeSource.PersistentVolumeClaim, w)
		case volume.VolumeSource.Ephemeral != nil:
			printEphemeralVolumeSource(volume.VolumeSource.Ephemeral, w)
		case volume.VolumeSource.RBD != nil:
			printRBDVolumeSource(volume.VolumeSource.RBD, w)
		case volume.VolumeSource.Quobyte != nil:
			printQuobyteVolumeSource(volume.VolumeSource.Quobyte, w)
		case volume.VolumeSource.DownwardAPI != nil:
			printDownwardAPIVolumeSource(volume.VolumeSource.DownwardAPI, w)
		case volume.VolumeSource.AzureDisk != nil:
			printAzureDiskVolumeSource(volume.VolumeSource.AzureDisk, w)
		case volume.VolumeSource.VsphereVolume != nil:
			printVsphereVolumeSource(volume.VolumeSource.VsphereVolume, w)
		case volume.VolumeSource.Cinder != nil:
			printCinderVolumeSource(volume.VolumeSource.Cinder, w)
		case volume.VolumeSource.PhotonPersistentDisk != nil:
			printPhotonPersistentDiskVolumeSource(volume.VolumeSource.PhotonPersistentDisk, w)
		case volume.VolumeSource.PortworxVolume != nil:
			printPortworxVolumeSource(volume.VolumeSource.PortworxVolume, w)
		case volume.VolumeSource.ScaleIO != nil:
			printScaleIOVolumeSource(volume.VolumeSource.ScaleIO, w)
		case volume.VolumeSource.CephFS != nil:
			printCephFSVolumeSource(volume.VolumeSource.CephFS, w)
		case volume.VolumeSource.StorageOS != nil:
			printStorageOSVolumeSource(volume.VolumeSource.StorageOS, w)
		case volume.VolumeSource.FC != nil:
			printFCVolumeSource(volume.VolumeSource.FC, w)
		case volume.VolumeSource.AzureFile != nil:
			printAzureFileVolumeSource(volume.VolumeSource.AzureFile, w)
		case volume.VolumeSource.FlexVolume != nil:
			printFlexVolumeSource(volume.VolumeSource.FlexVolume, w)
		case volume.VolumeSource.Flocker != nil:
			printFlockerVolumeSource(volume.VolumeSource.Flocker, w)
		case volume.VolumeSource.Projected != nil:
			printProjectedVolumeSource(volume.VolumeSource.Projected, w)
		case volume.VolumeSource.CSI != nil:
			printCSIVolumeSource(volume.VolumeSource.CSI, w)
		default:
			w.Write(LEVEL_1, "<unknown>\n")
		}
	}
}

func printHostPathVolumeSource(hostPath *corev1.HostPathVolumeSource, w PrefixWriter) {
	hostPathType := "<none>"
	if hostPath.Type != nil {
		hostPathType = string(*hostPath.Type)
	}
	w.Write(LEVEL_2, "Type:\tHostPath (bare host directory volume)\n"+
		"    Path:\t%v\n"+
		"    HostPathType:\t%v\n",
		hostPath.Path, hostPathType)
}

func printEmptyDirVolumeSource(emptyDir *corev1.EmptyDirVolumeSource, w PrefixWriter) {
	var sizeLimit string
	if emptyDir.SizeLimit != nil && emptyDir.SizeLimit.Cmp(resource.Quantity{}) > 0 {
		sizeLimit = fmt.Sprintf("%v", emptyDir.SizeLimit)
	} else {
		sizeLimit = "<unset>"
	}
	w.Write(LEVEL_2, "Type:\tEmptyDir (a temporary directory that shares a pod's lifetime)\n"+
		"    Medium:\t%v\n"+
		"    SizeLimit:\t%v\n",
		emptyDir.Medium, sizeLimit)
}

func printGCEPersistentDiskVolumeSource(gce *corev1.GCEPersistentDiskVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tGCEPersistentDisk (a Persistent Disk resource in Google Compute Engine)\n"+
		"    PDName:\t%v\n"+
		"    FSType:\t%v\n"+
		"    Partition:\t%v\n"+
		"    ReadOnly:\t%v\n",
		gce.PDName, gce.FSType, gce.Partition, gce.ReadOnly)
}

func printAWSElasticBlockStoreVolumeSource(aws *corev1.AWSElasticBlockStoreVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tAWSElasticBlockStore (a Persistent Disk resource in AWS)\n"+
		"    VolumeID:\t%v\n"+
		"    FSType:\t%v\n"+
		"    Partition:\t%v\n"+
		"    ReadOnly:\t%v\n",
		aws.VolumeID, aws.FSType, aws.Partition, aws.ReadOnly)
}

func printGitRepoVolumeSource(git *corev1.GitRepoVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tGitRepo (a volume that is pulled from git when the pod is created)\n"+
		"    Repository:\t%v\n"+
		"    Revision:\t%v\n",
		git.Repository, git.Revision)
}

func printSecretVolumeSource(secret *corev1.SecretVolumeSource, w PrefixWriter) {
	optional := secret.Optional != nil && *secret.Optional
	w.Write(LEVEL_2, "Type:\tSecret (a volume populated by a Secret)\n"+
		"    SecretName:\t%v\n"+
		"    Optional:\t%v\n",
		secret.SecretName, optional)
}

func printConfigMapVolumeSource(configMap *corev1.ConfigMapVolumeSource, w PrefixWriter) {
	optional := configMap.Optional != nil && *configMap.Optional
	w.Write(LEVEL_2, "Type:\tConfigMap (a volume populated by a ConfigMap)\n"+
		"    Name:\t%v\n"+
		"    Optional:\t%v\n",
		configMap.Name, optional)
}

func printProjectedVolumeSource(projected *corev1.ProjectedVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tProjected (a volume that contains injected data from multiple sources)\n")
	for _, source := range projected.Sources {
		if source.Secret != nil {
			w.Write(LEVEL_2, "SecretName:\t%v\n"+
				"    SecretOptionalName:\t%v\n",
				source.Secret.Name, source.Secret.Optional)
		} else if source.DownwardAPI != nil {
			w.Write(LEVEL_2, "DownwardAPI:\ttrue\n")
		} else if source.ConfigMap != nil {
			w.Write(LEVEL_2, "ConfigMapName:\t%v\n"+
				"    ConfigMapOptional:\t%v\n",
				source.ConfigMap.Name, source.ConfigMap.Optional)
		} else if source.ServiceAccountToken != nil {
			w.Write(LEVEL_2, "TokenExpirationSeconds:\t%d\n",
				*source.ServiceAccountToken.ExpirationSeconds)
		}
	}
}

func printNFSVolumeSource(nfs *corev1.NFSVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tNFS (an NFS mount that lasts the lifetime of a pod)\n"+
		"    Server:\t%v\n"+
		"    Path:\t%v\n"+
		"    ReadOnly:\t%v\n",
		nfs.Server, nfs.Path, nfs.ReadOnly)
}

func printQuobyteVolumeSource(quobyte *corev1.QuobyteVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tQuobyte (a Quobyte mount on the host that shares a pod's lifetime)\n"+
		"    Registry:\t%v\n"+
		"    Volume:\t%v\n"+
		"    ReadOnly:\t%v\n",
		quobyte.Registry, quobyte.Volume, quobyte.ReadOnly)
}

func printPortworxVolumeSource(pwxVolume *corev1.PortworxVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tPortworxVolume (a Portworx Volume resource)\n"+
		"    VolumeID:\t%v\n",
		pwxVolume.VolumeID)
}

func printISCSIVolumeSource(iscsi *corev1.ISCSIVolumeSource, w PrefixWriter) {
	initiator := "<none>"
	if iscsi.InitiatorName != nil {
		initiator = *iscsi.InitiatorName
	}
	w.Write(LEVEL_2, "Type:\tISCSI (an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod)\n"+
		"    TargetPortal:\t%v\n"+
		"    IQN:\t%v\n"+
		"    Lun:\t%v\n"+
		"    ISCSIInterface\t%v\n"+
		"    FSType:\t%v\n"+
		"    ReadOnly:\t%v\n"+
		"    Portals:\t%v\n"+
		"    DiscoveryCHAPAuth:\t%v\n"+
		"    SessionCHAPAuth:\t%v\n"+
		"    SecretRef:\t%v\n"+
		"    InitiatorName:\t%v\n",
		iscsi.TargetPortal, iscsi.IQN, iscsi.Lun, iscsi.ISCSIInterface, iscsi.FSType, iscsi.ReadOnly, iscsi.Portals, iscsi.DiscoveryCHAPAuth, iscsi.SessionCHAPAuth, iscsi.SecretRef, initiator)
}

func printISCSIPersistentVolumeSource(iscsi *corev1.ISCSIPersistentVolumeSource, w PrefixWriter) {
	initiatorName := "<none>"
	if iscsi.InitiatorName != nil {
		initiatorName = *iscsi.InitiatorName
	}
	w.Write(LEVEL_2, "Type:\tISCSI (an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod)\n"+
		"    TargetPortal:\t%v\n"+
		"    IQN:\t%v\n"+
		"    Lun:\t%v\n"+
		"    ISCSIInterface\t%v\n"+
		"    FSType:\t%v\n"+
		"    ReadOnly:\t%v\n"+
		"    Portals:\t%v\n"+
		"    DiscoveryCHAPAuth:\t%v\n"+
		"    SessionCHAPAuth:\t%v\n"+
		"    SecretRef:\t%v\n"+
		"    InitiatorName:\t%v\n",
		iscsi.TargetPortal, iscsi.IQN, iscsi.Lun, iscsi.ISCSIInterface, iscsi.FSType, iscsi.ReadOnly, iscsi.Portals, iscsi.DiscoveryCHAPAuth, iscsi.SessionCHAPAuth, iscsi.SecretRef, initiatorName)
}

func printGlusterfsVolumeSource(glusterfs *corev1.GlusterfsVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tGlusterfs (a Glusterfs mount on the host that shares a pod's lifetime)\n"+
		"    EndpointsName:\t%v\n"+
		"    Path:\t%v\n"+
		"    ReadOnly:\t%v\n",
		glusterfs.EndpointsName, glusterfs.Path, glusterfs.ReadOnly)
}

func printGlusterfsPersistentVolumeSource(glusterfs *corev1.GlusterfsPersistentVolumeSource, w PrefixWriter) {
	endpointsNamespace := "<unset>"
	if glusterfs.EndpointsNamespace != nil {
		endpointsNamespace = *glusterfs.EndpointsNamespace
	}
	w.Write(LEVEL_2, "Type:\tGlusterfs (a Glusterfs mount on the host that shares a pod's lifetime)\n"+
		"    EndpointsName:\t%v\n"+
		"    EndpointsNamespace:\t%v\n"+
		"    Path:\t%v\n"+
		"    ReadOnly:\t%v\n",
		glusterfs.EndpointsName, endpointsNamespace, glusterfs.Path, glusterfs.ReadOnly)
}

func printPersistentVolumeClaimVolumeSource(claim *corev1.PersistentVolumeClaimVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tPersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)\n"+
		"    ClaimName:\t%v\n"+
		"    ReadOnly:\t%v\n",
		claim.ClaimName, claim.ReadOnly)
}

func printEphemeralVolumeSource(ephemeral *corev1.EphemeralVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tEphemeralVolume (an inline specification for a volume that gets created and deleted with the pod)\n")
	if ephemeral.VolumeClaimTemplate != nil {
		printPersistentVolumeClaim(NewNestedPrefixWriter(w, LEVEL_2),
			&corev1.PersistentVolumeClaim{
				ObjectMeta: ephemeral.VolumeClaimTemplate.ObjectMeta,
				Spec:       ephemeral.VolumeClaimTemplate.Spec,
			}, false /* not a full PVC */)
	}
}

func printRBDVolumeSource(rbd *corev1.RBDVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tRBD (a Rados Block Device mount on the host that shares a pod's lifetime)\n"+
		"    CephMonitors:\t%v\n"+
		"    RBDImage:\t%v\n"+
		"    FSType:\t%v\n"+
		"    RBDPool:\t%v\n"+
		"    RadosUser:\t%v\n"+
		"    Keyring:\t%v\n"+
		"    SecretRef:\t%v\n"+
		"    ReadOnly:\t%v\n",
		rbd.CephMonitors, rbd.RBDImage, rbd.FSType, rbd.RBDPool, rbd.RadosUser, rbd.Keyring, rbd.SecretRef, rbd.ReadOnly)
}

func printRBDPersistentVolumeSource(rbd *corev1.RBDPersistentVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tRBD (a Rados Block Device mount on the host that shares a pod's lifetime)\n"+
		"    CephMonitors:\t%v\n"+
		"    RBDImage:\t%v\n"+
		"    FSType:\t%v\n"+
		"    RBDPool:\t%v\n"+
		"    RadosUser:\t%v\n"+
		"    Keyring:\t%v\n"+
		"    SecretRef:\t%v\n"+
		"    ReadOnly:\t%v\n",
		rbd.CephMonitors, rbd.RBDImage, rbd.FSType, rbd.RBDPool, rbd.RadosUser, rbd.Keyring, rbd.SecretRef, rbd.ReadOnly)
}

func printDownwardAPIVolumeSource(d *corev1.DownwardAPIVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tDownwardAPI (a volume populated by information about the pod)\n    Items:\n")
	for _, mapping := range d.Items {
		if mapping.FieldRef != nil {
			w.Write(LEVEL_3, "%v -> %v\n", mapping.FieldRef.FieldPath, mapping.Path)
		}
		if mapping.ResourceFieldRef != nil {
			w.Write(LEVEL_3, "%v -> %v\n", mapping.ResourceFieldRef.Resource, mapping.Path)
		}
	}
}

func printAzureDiskVolumeSource(d *corev1.AzureDiskVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tAzureDisk (an Azure Data Disk mount on the host and bind mount to the pod)\n"+
		"    DiskName:\t%v\n"+
		"    DiskURI:\t%v\n"+
		"    Kind: \t%v\n"+
		"    FSType:\t%v\n"+
		"    CachingMode:\t%v\n"+
		"    ReadOnly:\t%v\n",
		d.DiskName, d.DataDiskURI, *d.Kind, *d.FSType, *d.CachingMode, *d.ReadOnly)
}

func printVsphereVolumeSource(vsphere *corev1.VsphereVirtualDiskVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tvSphereVolume (a Persistent Disk resource in vSphere)\n"+
		"    VolumePath:\t%v\n"+
		"    FSType:\t%v\n"+
		"    StoragePolicyName:\t%v\n",
		vsphere.VolumePath, vsphere.FSType, vsphere.StoragePolicyName)
}

func printPhotonPersistentDiskVolumeSource(photon *corev1.PhotonPersistentDiskVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tPhotonPersistentDisk (a Persistent Disk resource in photon platform)\n"+
		"    PdID:\t%v\n"+
		"    FSType:\t%v\n",
		photon.PdID, photon.FSType)
}

func printCinderVolumeSource(cinder *corev1.CinderVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tCinder (a Persistent Disk resource in OpenStack)\n"+
		"    VolumeID:\t%v\n"+
		"    FSType:\t%v\n"+
		"    ReadOnly:\t%v\n"+
		"    SecretRef:\t%v\n",
		cinder.VolumeID, cinder.FSType, cinder.ReadOnly, cinder.SecretRef)
}

func printCinderPersistentVolumeSource(cinder *corev1.CinderPersistentVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tCinder (a Persistent Disk resource in OpenStack)\n"+
		"    VolumeID:\t%v\n"+
		"    FSType:\t%v\n"+
		"    ReadOnly:\t%v\n"+
		"    SecretRef:\t%v\n",
		cinder.VolumeID, cinder.FSType, cinder.ReadOnly, cinder.SecretRef)
}

func printScaleIOVolumeSource(sio *corev1.ScaleIOVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tScaleIO (a persistent volume backed by a block device in ScaleIO)\n"+
		"    Gateway:\t%v\n"+
		"    System:\t%v\n"+
		"    Protection Domain:\t%v\n"+
		"    Storage Pool:\t%v\n"+
		"    Storage Mode:\t%v\n"+
		"    VolumeName:\t%v\n"+
		"    FSType:\t%v\n"+
		"    ReadOnly:\t%v\n",
		sio.Gateway, sio.System, sio.ProtectionDomain, sio.StoragePool, sio.StorageMode, sio.VolumeName, sio.FSType, sio.ReadOnly)
}

func printScaleIOPersistentVolumeSource(sio *corev1.ScaleIOPersistentVolumeSource, w PrefixWriter) {
	var secretNS, secretName string
	if sio.SecretRef != nil {
		secretName = sio.SecretRef.Name
		secretNS = sio.SecretRef.Namespace
	}
	w.Write(LEVEL_2, "Type:\tScaleIO (a persistent volume backed by a block device in ScaleIO)\n"+
		"    Gateway:\t%v\n"+
		"    System:\t%v\n"+
		"    Protection Domain:\t%v\n"+
		"    Storage Pool:\t%v\n"+
		"    Storage Mode:\t%v\n"+
		"    VolumeName:\t%v\n"+
		"    SecretName:\t%v\n"+
		"    SecretNamespace:\t%v\n"+
		"    FSType:\t%v\n"+
		"    ReadOnly:\t%v\n",
		sio.Gateway, sio.System, sio.ProtectionDomain, sio.StoragePool, sio.StorageMode, sio.VolumeName, secretName, secretNS, sio.FSType, sio.ReadOnly)
}

func printLocalVolumeSource(ls *corev1.LocalVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tLocalVolume (a persistent volume backed by local storage on a node)\n"+
		"    Path:\t%v\n",
		ls.Path)
}

func printCephFSVolumeSource(cephfs *corev1.CephFSVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tCephFS (a CephFS mount on the host that shares a pod's lifetime)\n"+
		"    Monitors:\t%v\n"+
		"    Path:\t%v\n"+
		"    User:\t%v\n"+
		"    SecretFile:\t%v\n"+
		"    SecretRef:\t%v\n"+
		"    ReadOnly:\t%v\n",
		cephfs.Monitors, cephfs.Path, cephfs.User, cephfs.SecretFile, cephfs.SecretRef, cephfs.ReadOnly)
}

func printCephFSPersistentVolumeSource(cephfs *corev1.CephFSPersistentVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tCephFS (a CephFS mount on the host that shares a pod's lifetime)\n"+
		"    Monitors:\t%v\n"+
		"    Path:\t%v\n"+
		"    User:\t%v\n"+
		"    SecretFile:\t%v\n"+
		"    SecretRef:\t%v\n"+
		"    ReadOnly:\t%v\n",
		cephfs.Monitors, cephfs.Path, cephfs.User, cephfs.SecretFile, cephfs.SecretRef, cephfs.ReadOnly)
}

func printStorageOSVolumeSource(storageos *corev1.StorageOSVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tStorageOS (a StorageOS Persistent Disk resource)\n"+
		"    VolumeName:\t%v\n"+
		"    VolumeNamespace:\t%v\n"+
		"    FSType:\t%v\n"+
		"    ReadOnly:\t%v\n",
		storageos.VolumeName, storageos.VolumeNamespace, storageos.FSType, storageos.ReadOnly)
}

func printStorageOSPersistentVolumeSource(storageos *corev1.StorageOSPersistentVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tStorageOS (a StorageOS Persistent Disk resource)\n"+
		"    VolumeName:\t%v\n"+
		"    VolumeNamespace:\t%v\n"+
		"    FSType:\t%v\n"+
		"    ReadOnly:\t%v\n",
		storageos.VolumeName, storageos.VolumeNamespace, storageos.FSType, storageos.ReadOnly)
}

func printFCVolumeSource(fc *corev1.FCVolumeSource, w PrefixWriter) {
	lun := "<none>"
	if fc.Lun != nil {
		lun = strconv.Itoa(int(*fc.Lun))
	}
	w.Write(LEVEL_2, "Type:\tFC (a Fibre Channel disk)\n"+
		"    TargetWWNs:\t%v\n"+
		"    LUN:\t%v\n"+
		"    FSType:\t%v\n"+
		"    ReadOnly:\t%v\n",
		strings.Join(fc.TargetWWNs, ", "), lun, fc.FSType, fc.ReadOnly)
}

func printAzureFileVolumeSource(azureFile *corev1.AzureFileVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tAzureFile (an Azure File Service mount on the host and bind mount to the pod)\n"+
		"    SecretName:\t%v\n"+
		"    ShareName:\t%v\n"+
		"    ReadOnly:\t%v\n",
		azureFile.SecretName, azureFile.ShareName, azureFile.ReadOnly)
}

func printAzureFilePersistentVolumeSource(azureFile *corev1.AzureFilePersistentVolumeSource, w PrefixWriter) {
	ns := ""
	if azureFile.SecretNamespace != nil {
		ns = *azureFile.SecretNamespace
	}
	w.Write(LEVEL_2, "Type:\tAzureFile (an Azure File Service mount on the host and bind mount to the pod)\n"+
		"    SecretName:\t%v\n"+
		"    SecretNamespace:\t%v\n"+
		"    ShareName:\t%v\n"+
		"    ReadOnly:\t%v\n",
		azureFile.SecretName, ns, azureFile.ShareName, azureFile.ReadOnly)
}

func printFlexPersistentVolumeSource(flex *corev1.FlexPersistentVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tFlexVolume (a generic volume resource that is provisioned/attached using an exec based plugin)\n"+
		"    Driver:\t%v\n"+
		"    FSType:\t%v\n"+
		"    SecretRef:\t%v\n"+
		"    ReadOnly:\t%v\n"+
		"    Options:\t%v\n",
		flex.Driver, flex.FSType, flex.SecretRef, flex.ReadOnly, flex.Options)
}

func printFlexVolumeSource(flex *corev1.FlexVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tFlexVolume (a generic volume resource that is provisioned/attached using an exec based plugin)\n"+
		"    Driver:\t%v\n"+
		"    FSType:\t%v\n"+
		"    SecretRef:\t%v\n"+
		"    ReadOnly:\t%v\n"+
		"    Options:\t%v\n",
		flex.Driver, flex.FSType, flex.SecretRef, flex.ReadOnly, flex.Options)
}

func printFlockerVolumeSource(flocker *corev1.FlockerVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tFlocker (a Flocker volume mounted by the Flocker agent)\n"+
		"    DatasetName:\t%v\n"+
		"    DatasetUUID:\t%v\n",
		flocker.DatasetName, flocker.DatasetUUID)
}

func printCSIVolumeSource(csi *corev1.CSIVolumeSource, w PrefixWriter) {
	var readOnly bool
	var fsType string
	if csi.ReadOnly != nil && *csi.ReadOnly {
		readOnly = true
	}
	if csi.FSType != nil {
		fsType = *csi.FSType
	}
	w.Write(LEVEL_2, "Type:\tCSI (a Container Storage Interface (CSI) volume source)\n"+
		"    Driver:\t%v\n"+
		"    FSType:\t%v\n"+
		"    ReadOnly:\t%v\n",
		csi.Driver, fsType, readOnly)
	printCSIPersistentVolumeAttributesMultiline(w, "VolumeAttributes", csi.VolumeAttributes)
}

func printCSIPersistentVolumeSource(csi *corev1.CSIPersistentVolumeSource, w PrefixWriter) {
	w.Write(LEVEL_2, "Type:\tCSI (a Container Storage Interface (CSI) volume source)\n"+
		"    Driver:\t%v\n"+
		"    FSType:\t%v\n"+
		"    VolumeHandle:\t%v\n"+
		"    ReadOnly:\t%v\n",
		csi.Driver, csi.FSType, csi.VolumeHandle, csi.ReadOnly)
	printCSIPersistentVolumeAttributesMultiline(w, "VolumeAttributes", csi.VolumeAttributes)
}

func printCSIPersistentVolumeAttributesMultiline(w PrefixWriter, title string, annotations map[string]string) {
	printCSIPersistentVolumeAttributesMultilineIndent(w, "", title, "\t", annotations, sets.NewString())
}

func printCSIPersistentVolumeAttributesMultilineIndent(w PrefixWriter, initialIndent, title, innerIndent string, attributes map[string]string, skip sets.String) {
	w.Write(LEVEL_2, "%s%s:%s", initialIndent, title, innerIndent)

	if len(attributes) == 0 {
		w.WriteLine("<none>")
		return
	}

	// to print labels in the sorted order
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		if skip.Has(key) {
			continue
		}
		keys = append(keys, key)
	}
	if len(attributes) == 0 {
		w.WriteLine("<none>")
		return
	}
	sort.Strings(keys)

	for i, key := range keys {
		if i != 0 {
			w.Write(LEVEL_2, initialIndent)
			w.Write(LEVEL_2, innerIndent)
		}
		line := fmt.Sprintf("%s=%s", key, attributes[key])
		if len(line) > maxAnnotationLen {
			w.Write(LEVEL_2, "%s...\n", line[:maxAnnotationLen])
		} else {
			w.Write(LEVEL_2, "%s\n", line)
		}
	}
}

// printPodTolerationsMultiline prints multiple tolerations with a proper alignment.
func printPodTolerationsMultiline(w PrefixWriter, title string, tolerations []corev1.Toleration) {
	printTolerationsMultilineWithIndent(w, "", title, "\t", tolerations)
}

// printTolerationsMultilineWithIndent prints multiple tolerations with a user-defined alignment.
func printTolerationsMultilineWithIndent(w PrefixWriter, initialIndent, title, innerIndent string, tolerations []corev1.Toleration) {
	w.Write(LEVEL_0, "%s%s:%s", initialIndent, title, innerIndent)

	if len(tolerations) == 0 {
		w.WriteLine("<none>")
		return
	}

	// to print tolerations in the sorted order
	sort.Slice(tolerations, func(i, j int) bool {
		return tolerations[i].Key < tolerations[j].Key
	})

	for i, toleration := range tolerations {
		if i != 0 {
			w.Write(LEVEL_0, "%s", initialIndent)
			w.Write(LEVEL_0, "%s", innerIndent)
		}
		w.Write(LEVEL_0, "%s", toleration.Key)
		if len(toleration.Value) != 0 {
			w.Write(LEVEL_0, "=%s", toleration.Value)
		}
		if len(toleration.Effect) != 0 {
			w.Write(LEVEL_0, ":%s", toleration.Effect)
		}
		// tolerations:
		// - operator: "Exists"
		// is a special case which tolerates everything
		if toleration.Operator == corev1.TolerationOpExists && len(toleration.Value) == 0 {
			if len(toleration.Key) != 0 || len(toleration.Effect) != 0 {
				w.Write(LEVEL_0, " op=Exists")
			} else {
				w.Write(LEVEL_0, "op=Exists")
			}
		}

		if toleration.TolerationSeconds != nil {
			w.Write(LEVEL_0, " for %ds", *toleration.TolerationSeconds)
		}
		w.Write(LEVEL_0, "\n")
	}
}

func stringOrNone(s string) string {
	return stringOrDefaultValue(s, "<none>")
}

func stringOrDefaultValue(s, defaultValue string) string {
	if len(s) > 0 {
		return s
	}
	return defaultValue
}

func describeStatus(stateName string, state corev1.ContainerState, w PrefixWriter) {
	switch {
	case state.Running != nil:
		w.Write(LEVEL_2, "%s:\tRunning\n", stateName)
		w.Write(LEVEL_3, "Started:\t%v\n", state.Running.StartedAt.Time.Format(time.RFC1123Z))
	case state.Waiting != nil:
		w.Write(LEVEL_2, "%s:\tWaiting\n", stateName)
		if state.Waiting.Reason != "" {
			w.Write(LEVEL_3, "Reason:\t%s\n", state.Waiting.Reason)
		}
	case state.Terminated != nil:
		w.Write(LEVEL_2, "%s:\tTerminated\n", stateName)
		if state.Terminated.Reason != "" {
			w.Write(LEVEL_3, "Reason:\t%s\n", state.Terminated.Reason)
		}
		if state.Terminated.Message != "" {
			w.Write(LEVEL_3, "Message:\t%s\n", state.Terminated.Message)
		}
		w.Write(LEVEL_3, "Exit Code:\t%d\n", state.Terminated.ExitCode)
		if state.Terminated.Signal > 0 {
			w.Write(LEVEL_3, "Signal:\t%d\n", state.Terminated.Signal)
		}
		w.Write(LEVEL_3, "Started:\t%s\n", state.Terminated.StartedAt.Time.Format(time.RFC1123Z))
		w.Write(LEVEL_3, "Finished:\t%s\n", state.Terminated.FinishedAt.Time.Format(time.RFC1123Z))
	default:
		w.Write(LEVEL_2, "%s:\tWaiting\n", stateName)
	}
}

func describeTopologySpreadConstraints(tscs []corev1.TopologySpreadConstraint, w PrefixWriter, space string) {
	if len(tscs) == 0 {
		return
	}

	sort.Slice(tscs, func(i, j int) bool {
		return tscs[i].TopologyKey < tscs[j].TopologyKey
	})

	w.Write(LEVEL_0, "%sTopology Spread Constraints:\t", space)
	for i, tsc := range tscs {
		if i != 0 {
			w.Write(LEVEL_0, "%s", space)
			w.Write(LEVEL_0, "%s", "\t")
		}

		w.Write(LEVEL_0, "%s:", tsc.TopologyKey)
		w.Write(LEVEL_0, "%v", tsc.WhenUnsatisfiable)
		w.Write(LEVEL_0, " when max skew %d is exceeded", tsc.MaxSkew)
		if tsc.LabelSelector != nil {
			w.Write(LEVEL_0, " for selector %s", metav1.FormatLabelSelector(tsc.LabelSelector))
		}
		w.Write(LEVEL_0, "\n")
	}
}

func printBool(value bool) string {
	if value {
		return "True"
	}

	return "False"
}

// printPersistentVolumeClaim is used for both PVCs and PersistentVolumeClaimTemplate. For the latter,
// we need to skip some fields which have no meaning.
func printPersistentVolumeClaim(w PrefixWriter, pvc *corev1.PersistentVolumeClaim, isFullPVC bool) {
	if isFullPVC {
		w.Write(LEVEL_0, "Name:\t%s\n", pvc.Name)
		w.Write(LEVEL_0, "Namespace:\t%s\n", pvc.Namespace)
	}
	w.Write(LEVEL_0, "StorageClass:\t%s\n", storageutil.GetPersistentVolumeClaimClass(pvc))
	if isFullPVC {
		if pvc.ObjectMeta.DeletionTimestamp != nil {
			w.Write(LEVEL_0, "Status:\tTerminating (lasts %s)\n", translateTimestampSince(*pvc.ObjectMeta.DeletionTimestamp))
		} else {
			w.Write(LEVEL_0, "Status:\t%v\n", pvc.Status.Phase)
		}
	}
	w.Write(LEVEL_0, "Volume:\t%s\n", pvc.Spec.VolumeName)
	printLabelsMultiline(w, "Labels", pvc.Labels)
	printAnnotationsMultiline(w, "Annotations", pvc.Annotations)
	if isFullPVC {
		w.Write(LEVEL_0, "Finalizers:\t%v\n", pvc.ObjectMeta.Finalizers)
	}
	storage := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	capacity := ""
	accessModes := ""
	if pvc.Spec.VolumeName != "" {
		accessModes = storageutil.GetAccessModesAsString(pvc.Status.AccessModes)
		storage = pvc.Status.Capacity[corev1.ResourceStorage]
		capacity = storage.String()
	}
	w.Write(LEVEL_0, "Capacity:\t%s\n", capacity)
	w.Write(LEVEL_0, "Access Modes:\t%s\n", accessModes)
	if pvc.Spec.VolumeMode != nil {
		w.Write(LEVEL_0, "VolumeMode:\t%v\n", *pvc.Spec.VolumeMode)
	}
	if pvc.Spec.DataSource != nil {
		w.Write(LEVEL_0, "DataSource:\n")
		if pvc.Spec.DataSource.APIGroup != nil {
			w.Write(LEVEL_1, "APIGroup:\t%v\n", *pvc.Spec.DataSource.APIGroup)
		}
		w.Write(LEVEL_1, "Kind:\t%v\n", pvc.Spec.DataSource.Kind)
		w.Write(LEVEL_1, "Name:\t%v\n", pvc.Spec.DataSource.Name)
	}
}

func describeService(service *corev1.Service) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", service.Name)
		w.Write(LEVEL_0, "Namespace:\t%s\n", service.Namespace)
		printLabelsMultiline(w, "Labels", service.Labels)
		printAnnotationsMultiline(w, "Annotations", service.Annotations)
		w.Write(LEVEL_0, "Selector:\t%s\n", labels.FormatLabels(service.Spec.Selector))
		w.Write(LEVEL_0, "Type:\t%s\n", service.Spec.Type)

		if service.Spec.IPFamilyPolicy != nil {
			w.Write(LEVEL_0, "IP Family Policy:\t%s\n", *(service.Spec.IPFamilyPolicy))
		}

		if len(service.Spec.IPFamilies) > 0 {
			ipfamiliesStrings := make([]string, 0, len(service.Spec.IPFamilies))
			for _, family := range service.Spec.IPFamilies {
				ipfamiliesStrings = append(ipfamiliesStrings, string(family))
			}

			w.Write(LEVEL_0, "IP Families:\t%s\n", strings.Join(ipfamiliesStrings, ","))
		} else {
			w.Write(LEVEL_0, "IP Families:\t%s\n", "<none>")
		}

		w.Write(LEVEL_0, "IP:\t%s\n", service.Spec.ClusterIP)
		if len(service.Spec.ClusterIPs) > 0 {
			w.Write(LEVEL_0, "IPs:\t%s\n", strings.Join(service.Spec.ClusterIPs, ","))
		} else {
			w.Write(LEVEL_0, "IPs:\t%s\n", "<none>")
		}

		if len(service.Spec.ExternalIPs) > 0 {
			w.Write(LEVEL_0, "External IPs:\t%v\n", strings.Join(service.Spec.ExternalIPs, ","))
		}
		if service.Spec.LoadBalancerIP != "" {
			w.Write(LEVEL_0, "IP:\t%s\n", service.Spec.LoadBalancerIP)
		}
		if service.Spec.ExternalName != "" {
			w.Write(LEVEL_0, "External Name:\t%s\n", service.Spec.ExternalName)
		}
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			list := buildIngressString(service.Status.LoadBalancer.Ingress)
			w.Write(LEVEL_0, "LoadBalancer Ingress:\t%s\n", list)
		}
		for i := range service.Spec.Ports {
			sp := &service.Spec.Ports[i]

			name := sp.Name
			if name == "" {
				name = "<unset>"
			}
			w.Write(LEVEL_0, "Port:\t%s\t%d/%s\n", name, sp.Port, sp.Protocol)
			if sp.TargetPort.Type == intstr.Type(intstr.Int) {
				w.Write(LEVEL_0, "TargetPort:\t%d/%s\n", sp.TargetPort.IntVal, sp.Protocol)
			} else {
				w.Write(LEVEL_0, "TargetPort:\t%s/%s\n", sp.TargetPort.StrVal, sp.Protocol)
			}
			if sp.NodePort != 0 {
				w.Write(LEVEL_0, "NodePort:\t%s\t%d/%s\n", name, sp.NodePort, sp.Protocol)
			}
			// w.Write(LEVEL_0, "Endpoints:\t%s\n", formatEndpoints(endpoints, sets.NewString(sp.Name)))
		}
		w.Write(LEVEL_0, "Session Affinity:\t%s\n", service.Spec.SessionAffinity)
		if service.Spec.ExternalTrafficPolicy != "" {
			w.Write(LEVEL_0, "External Traffic Policy:\t%s\n", service.Spec.ExternalTrafficPolicy)
		}
		if service.Spec.HealthCheckNodePort != 0 {
			w.Write(LEVEL_0, "HealthCheck NodePort:\t%d\n", service.Spec.HealthCheckNodePort)
		}
		if len(service.Spec.LoadBalancerSourceRanges) > 0 {
			w.Write(LEVEL_0, "LoadBalancer Source Ranges:\t%v\n", strings.Join(service.Spec.LoadBalancerSourceRanges, ","))
		}
		return nil
	})
}

func buildIngressString(ingress []corev1.LoadBalancerIngress) string {
	var buffer bytes.Buffer

	for i := range ingress {
		if i != 0 {
			buffer.WriteString(", ")
		}
		if ingress[i].IP != "" {
			buffer.WriteString(ingress[i].IP)
		} else {
			buffer.WriteString(ingress[i].Hostname)
		}
	}
	return buffer.String()
}

func describeDeployment(d *appsv1.Deployment) (string, error) {
	selector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return "", err
	}
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", d.ObjectMeta.Name)
		w.Write(LEVEL_0, "Namespace:\t%s\n", d.ObjectMeta.Namespace)
		w.Write(LEVEL_0, "CreationTimestamp:\t%s\n", d.CreationTimestamp.Time.Format(time.RFC1123Z))
		printLabelsMultiline(w, "Labels", d.Labels)
		printAnnotationsMultiline(w, "Annotations", d.Annotations)
		w.Write(LEVEL_0, "Selector:\t%s\n", selector)
		w.Write(LEVEL_0, "Replicas:\t%d desired | %d updated | %d total | %d available | %d unavailable\n", *(d.Spec.Replicas), d.Status.UpdatedReplicas, d.Status.Replicas, d.Status.AvailableReplicas, d.Status.UnavailableReplicas)
		w.Write(LEVEL_0, "StrategyType:\t%s\n", d.Spec.Strategy.Type)
		w.Write(LEVEL_0, "MinReadySeconds:\t%d\n", d.Spec.MinReadySeconds)
		if d.Spec.Strategy.RollingUpdate != nil {
			ru := d.Spec.Strategy.RollingUpdate
			w.Write(LEVEL_0, "RollingUpdateStrategy:\t%s max unavailable, %s max surge\n", ru.MaxUnavailable.String(), ru.MaxSurge.String())
		}
		DescribePodTemplate(&d.Spec.Template, w)
		if len(d.Status.Conditions) > 0 {
			w.Write(LEVEL_0, "Conditions:\n  Type\tStatus\tReason\n")
			w.Write(LEVEL_1, "----\t------\t------\n")
			for _, c := range d.Status.Conditions {
				w.Write(LEVEL_1, "%v \t%v\t%v\n", c.Type, c.Status, c.Reason)
			}
		}
		return nil
	})
}

func describeDaemonSet(daemon *appsv1.DaemonSet) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", daemon.Name)
		selector, err := metav1.LabelSelectorAsSelector(daemon.Spec.Selector)
		if err != nil {
			// this shouldn't happen if LabelSelector passed validation
			return err
		}
		w.Write(LEVEL_0, "Selector:\t%s\n", selector)
		w.Write(LEVEL_0, "Node-Selector:\t%s\n", labels.FormatLabels(daemon.Spec.Template.Spec.NodeSelector))
		printLabelsMultiline(w, "Labels", daemon.Labels)
		printAnnotationsMultiline(w, "Annotations", daemon.Annotations)
		w.Write(LEVEL_0, "Desired Number of Nodes Scheduled: %d\n", daemon.Status.DesiredNumberScheduled)
		w.Write(LEVEL_0, "Current Number of Nodes Scheduled: %d\n", daemon.Status.CurrentNumberScheduled)
		w.Write(LEVEL_0, "Number of Nodes Scheduled with Up-to-date Pods: %d\n", daemon.Status.UpdatedNumberScheduled)
		w.Write(LEVEL_0, "Number of Nodes Scheduled with Available Pods: %d\n", daemon.Status.NumberAvailable)
		w.Write(LEVEL_0, "Number of Nodes Misscheduled: %d\n", daemon.Status.NumberMisscheduled)
		// w.Write(LEVEL_0, "Pods Status:\t%d Running / %d Waiting / %d Succeeded / %d Failed\n", running, waiting, succeeded, failed)
		DescribePodTemplate(&daemon.Spec.Template, w)
		return nil
	})
}

func describeReplicaSet(rs *appsv1.ReplicaSet) (string, error) {
	return tabbedString(func(out io.Writer) error {
		w := NewPrefixWriter(out)
		w.Write(LEVEL_0, "Name:\t%s\n", rs.Name)
		w.Write(LEVEL_0, "Namespace:\t%s\n", rs.Namespace)
		w.Write(LEVEL_0, "Selector:\t%s\n", metav1.FormatLabelSelector(rs.Spec.Selector))
		printLabelsMultiline(w, "Labels", rs.Labels)
		printAnnotationsMultiline(w, "Annotations", rs.Annotations)
		if controlledBy := printController(rs); len(controlledBy) > 0 {
			w.Write(LEVEL_0, "Controlled By:\t%s\n", controlledBy)
		}
		w.Write(LEVEL_0, "Replicas:\t%d current / %d desired\n", rs.Status.Replicas, *rs.Spec.Replicas)
		// w.Write(LEVEL_0, "Pods Status:\t")
		// if getPodErr != nil {
		// 	w.Write(LEVEL_0, "error in fetching pods: %s\n", getPodErr)
		// } else {
		// 	w.Write(LEVEL_0, "%d Running / %d Waiting / %d Succeeded / %d Failed\n", running, waiting, succeeded, failed)
		// }
		DescribePodTemplate(&rs.Spec.Template, w)
		if len(rs.Status.Conditions) > 0 {
			w.Write(LEVEL_0, "Conditions:\n  Type\tStatus\tReason\n")
			w.Write(LEVEL_1, "----\t------\t------\n")
			for _, c := range rs.Status.Conditions {
				w.Write(LEVEL_1, "%v \t%v\t%v\n", c.Type, c.Status, c.Reason)
			}
		}
		return nil
	})
}
