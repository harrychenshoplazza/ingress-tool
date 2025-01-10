package api

import (
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
)

func SetupRouter(clientset *kubernetes.Clientset) *gin.Engine {
	r := gin.Default()
	v1 := r.Group("api/v1")
	{
		v1.GET("/ingresses", ListIngress(clientset))
		v1.POST("/eks/nodegroups", ListNodeGroups())
		v1.POST("/eks/multiacc-nodegroups", ListMultiAccNodeGroups())
		v1.POST("/eks/login", LoginEKS())
		v1.POST("/restartDeployment", RestartDeployment(clientset))
		v1.POST("/restartPod", RestartPod(clientset))
	}
	return r
}
