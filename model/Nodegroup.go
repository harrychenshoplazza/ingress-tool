package model

type NodeGroupRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	Region      string `json:"region" binding:"required"`
}

type RestartDeploymentReq struct {
	Namespace      string `json:"namespace" binding:"required"`
	DeploymentName string `json:"deploymentName" binding:"required"`
}

type RestartPodReq struct {
	Namespace string `json:"namespace" binding:"required"`
	PodName   string `json:"podName" binding:"required"`
}
