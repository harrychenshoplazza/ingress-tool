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
	}
	return r
}
