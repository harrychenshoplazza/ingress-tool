package api

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"log"
	"net/http"
	"os"
	"strings"

	"ingress-tool/model"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func InitK8sClient(kubeConfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		log.Fatalf("Fail to get kubeconfig: %v", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Fail to create kubernetes client: %v", err)
		return nil, err
	}

	return clientset, nil
}

func ListIngress(clientset *kubernetes.Clientset) gin.HandlerFunc {
	return func(c *gin.Context) {
		total := 0
		namespace := c.DefaultQuery("namespace", "")
		pathFilter := c.Query("path")
		serviceFilter := c.Query("service")

		ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var response []map[string]interface{}

		for _, ingress := range ingresses.Items {
			ingressInfo := map[string]interface{}{
				"namespace": ingress.Namespace,
				"name":      ingress.Name,
				"rules":     []map[string]interface{}{},
			}
			for _, rule := range ingress.Spec.Rules {
				ruleInfo := map[string]interface{}{
					"host":  rule.Host,
					"paths": []map[string]interface{}{},
				}
				if rule.HTTP != nil {
					for _, path := range rule.HTTP.Paths {
						if pathFilter != "" && !strings.EqualFold(path.Path, pathFilter) {
							continue
						}
						if serviceFilter != "" && !strings.EqualFold(path.Backend.Service.Name, serviceFilter) {
							continue
						}
						pathInfo := map[string]interface{}{
							"path":    path.Path,
							"service": path.Backend.Service.Name,
							"port":    path.Backend.Service.Port.Number,
						}
						ruleInfo["paths"] = append(ruleInfo["paths"].([]map[string]interface{}), pathInfo)
						total += 1
					}
				}
				ingressInfo["rules"] = append(ingressInfo["rules"].([]map[string]interface{}), ruleInfo)
			}
			rulesNotEmpty := len(ingressInfo["rules"].([]map[string]interface{})) > 0
			pathsNotEmpty := len(ingressInfo["rules"].([]map[string]interface{})[0]["paths"].([]map[string]interface{})) > 0
			if rulesNotEmpty && pathsNotEmpty {
				response = append(response, ingressInfo)
			}
		}
		fmt.Printf("Total ingress rules: %d\n", total)
		c.JSON(http.StatusOK, response)
	}
}

func ListNodeGroups() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.NodeGroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		nodegroups, err := getNodeGroups(req.ClusterName, req.Region)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"cluster_name": req.ClusterName,
			"region":       req.Region,
			"node_groups":  nodegroups,
		})
	}
}

func getNodeGroups(clusterName, region string) ([]map[string]interface{}, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := eks.NewFromConfig(cfg)
	nodegroups, err := getNodeGroupsAction(client, clusterName)
	if err != nil {
		return nil, err
	}
	return nodegroups, nil
}

func getNodeGroupsAction(client *eks.Client, clusterName string) ([]map[string]interface{}, error) {
	listNodeGroupsOutput, err := client.ListNodegroups(context.TODO(), &eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodegroups: %w", err)
	}

	var nodegroups []map[string]interface{}
	for _, nodegroup := range listNodeGroupsOutput.Nodegroups {
		describeNodeGroup, err := client.DescribeNodegroup(context.TODO(), &eks.DescribeNodegroupInput{
			ClusterName:   aws.String(clusterName),
			NodegroupName: aws.String(nodegroup),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe nodegroup [%s]: %w", nodegroup, err)
		}
		ngInfo := map[string]interface{}{
			"name":          aws.ToString(describeNodeGroup.Nodegroup.NodegroupName),
			"scalingConfig": describeNodeGroup.Nodegroup.ScalingConfig,
		}
		nodegroups = append(nodegroups, ngInfo)
	}
	return nodegroups, nil
}

func ListMultiAccNodeGroups() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.NodeGroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		profile := os.Getenv("HOME") + "/.aws/credentials"
		nodegroups, err := getMultiAccsNodegroups(req.ClusterName, req.Region, profile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"cluster_name": req.ClusterName,
			"region":       req.Region,
			"node_groups":  nodegroups,
		})
	}
}

func getMultiAccsNodegroups(clusterName, region, profile string) ([]map[string]interface{}, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(profile), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := eks.NewFromConfig(cfg)
	nodegroups, err := getNodeGroupsAction(client, clusterName)
	if err != nil {
		return nil, err
	}
	return nodegroups, nil
}

func LoginEKS() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.NodeGroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := loginToEKS(req.ClusterName, req.Region)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"cluster_name": req.ClusterName,
			"region":       req.Region,
		})
	}
}

func loginToEKS(clusterName, region string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := eks.NewFromConfig(cfg)
	clusterOutput, err := client.DescribeCluster(context.TODO(), &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		return fmt.Errorf("failed to describe EKS cluster: %w", err)
	}
	cluster := clusterOutput.Cluster

	kubeconfig := fmt.Sprintf(`
apiVersion: v1
clusters:
- cluster:
    server: %s
    certificate-authority-data: %s
  name: %s
contexts:
- context:
    cluster: %s
    user: eks-user
  name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: eks-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: aws
      args:
        - eks
        - get-token
        - --cluster-name
        - %s
        - --region
        - %s
      env: null`, cluster.Endpoint, *cluster.CertificateAuthority.Data, clusterName, clusterName, clusterName, clusterName,
		clusterName, region)

	kubeconfigPath := fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfig), 0644); err != nil {
		return fmt.Errorf("failed to write kubeconfig to file: %w", err)
	}
	fmt.Printf("Successfully updated kubeconfig for cluster: %s\n", clusterName)
	return nil
}
