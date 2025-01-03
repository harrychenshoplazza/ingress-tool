package model

type NodeGroupRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	Region      string `json:"region" binding:"required"`
}
