package api

import (
	"context"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/http"
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
		namespace := c.DefaultQuery("namespace", "")

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
						pathInfo := map[string]interface{}{
							"path":    path.Path,
							"service": path.Backend.Service.Name,
							"port":    path.Backend.Service.Port.Number,
						}
						ruleInfo["paths"] = append(ruleInfo["paths"].([]map[string]interface{}), pathInfo)
					}
				}
				ingressInfo["rules"] = append(ingressInfo["rules"].([]map[string]interface{}), ruleInfo)
			}
			response = append(response, ingressInfo)
		}
		c.JSON(http.StatusOK, response)
	}
}
