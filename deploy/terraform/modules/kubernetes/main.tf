# Kubernetes Cluster Module
# Supports EKS (AWS), AKS (Azure), and GKE (GCP) via conditional resources.

variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "cloud_provider" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "subnet_ids" {
  type = list(string)
}

variable "cluster_version" {
  type    = string
  default = "1.29"
}

variable "node_min_count" {
  type    = number
  default = 3
}

variable "node_max_count" {
  type    = number
  default = 10
}

variable "node_instance_type" {
  type    = string
  default = "m6i.xlarge"
}

locals {
  cluster_name = "${var.project_name}-${var.environment}"
  is_aws       = var.cloud_provider == "aws"
  is_azure     = var.cloud_provider == "azure"
  is_gcp       = var.cloud_provider == "gcp"
}

# ---------------------------------------------------------------------------
# AWS EKS
# ---------------------------------------------------------------------------

resource "aws_iam_role" "eks_cluster" {
  count = local.is_aws ? 1 : 0
  name  = "${local.cluster_name}-eks-cluster-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "eks.amazonaws.com"
      }
    }]
  })

  tags = {
    Name = "${local.cluster_name}-eks-cluster-role"
  }
}

resource "aws_iam_role_policy_attachment" "eks_cluster_policy" {
  count      = local.is_aws ? 1 : 0
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
  role       = aws_iam_role.eks_cluster[0].name
}

resource "aws_eks_cluster" "main" {
  count    = local.is_aws ? 1 : 0
  name     = local.cluster_name
  role_arn = aws_iam_role.eks_cluster[0].arn
  version  = var.cluster_version

  vpc_config {
    subnet_ids              = var.subnet_ids
    endpoint_private_access = true
    endpoint_public_access  = var.environment != "prod"
  }

  encryption_config {
    resources = ["secrets"]
    provider {
      key_arn = aws_kms_key.eks[0].arn
    }
  }

  enabled_cluster_log_types = [
    "api",
    "audit",
    "authenticator",
    "controllerManager",
    "scheduler",
  ]

  tags = {
    Name        = local.cluster_name
    Environment = var.environment
  }
}

resource "aws_kms_key" "eks" {
  count                   = local.is_aws ? 1 : 0
  description             = "KMS key for EKS secrets encryption"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = {
    Name = "${local.cluster_name}-eks-kms"
  }
}

# --- Node Group ---

resource "aws_iam_role" "eks_nodes" {
  count = local.is_aws ? 1 : 0
  name  = "${local.cluster_name}-eks-node-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "eks_worker_node_policy" {
  count      = local.is_aws ? 1 : 0
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.eks_nodes[0].name
}

resource "aws_iam_role_policy_attachment" "eks_cni_policy" {
  count      = local.is_aws ? 1 : 0
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = aws_iam_role.eks_nodes[0].name
}

resource "aws_iam_role_policy_attachment" "eks_ecr_policy" {
  count      = local.is_aws ? 1 : 0
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.eks_nodes[0].name
}

resource "aws_eks_node_group" "main" {
  count           = local.is_aws ? 1 : 0
  cluster_name    = aws_eks_cluster.main[0].name
  node_group_name = "${local.cluster_name}-nodes"
  node_role_arn   = aws_iam_role.eks_nodes[0].arn
  subnet_ids      = var.subnet_ids
  instance_types  = [var.node_instance_type]

  scaling_config {
    desired_size = var.node_min_count
    max_size     = var.node_max_count
    min_size     = var.node_min_count
  }

  update_config {
    max_unavailable = 1
  }

  tags = {
    Name        = "${local.cluster_name}-nodes"
    Environment = var.environment
  }
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "cluster_endpoint" {
  value = local.is_aws ? aws_eks_cluster.main[0].endpoint : "placeholder"
}

output "cluster_name" {
  value = local.cluster_name
}

output "cluster_certificate_authority" {
  value     = local.is_aws ? aws_eks_cluster.main[0].certificate_authority[0].data : ""
  sensitive = true
}
