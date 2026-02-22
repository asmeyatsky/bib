# ---------------------------------------------------------------------------
# General
# ---------------------------------------------------------------------------

variable "project_name" {
  description = "Project name used for resource naming and tagging"
  type        = string
  default     = "bib"
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod."
  }
}

variable "cloud_provider" {
  description = "Cloud provider: aws, azure, or gcp"
  type        = string
  default     = "aws"

  validation {
    condition     = contains(["aws", "azure", "gcp"], var.cloud_provider)
    error_message = "Cloud provider must be one of: aws, azure, gcp."
  }
}

variable "aws_region" {
  description = "AWS region for resource deployment"
  type        = string
  default     = "us-east-1"
}

# ---------------------------------------------------------------------------
# Networking
# ---------------------------------------------------------------------------

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

# ---------------------------------------------------------------------------
# Kubernetes
# ---------------------------------------------------------------------------

variable "kubernetes_version" {
  description = "Kubernetes cluster version"
  type        = string
  default     = "1.29"
}

variable "node_min_count" {
  description = "Minimum number of nodes in the Kubernetes cluster"
  type        = number
  default     = 3
}

variable "node_max_count" {
  description = "Maximum number of nodes in the Kubernetes cluster"
  type        = number
  default     = 10
}

variable "node_instance_type" {
  description = "Instance type for Kubernetes worker nodes"
  type        = string
  default     = "m6i.xlarge"
}

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------

variable "db_engine" {
  description = "Database engine (postgres)"
  type        = string
  default     = "postgres"
}

variable "db_version" {
  description = "Database engine version"
  type        = string
  default     = "16.2"
}

variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.r6g.xlarge"
}

variable "db_storage_gb" {
  description = "Allocated storage in GB"
  type        = number
  default     = 100
}

variable "db_multi_az" {
  description = "Enable multi-AZ deployment for the database"
  type        = bool
  default     = true
}

variable "db_backup_retention_days" {
  description = "Number of days to retain automated backups"
  type        = number
  default     = 30
}

# ---------------------------------------------------------------------------
# Kafka
# ---------------------------------------------------------------------------

variable "kafka_broker_count" {
  description = "Number of Kafka broker nodes"
  type        = number
  default     = 3
}

variable "kafka_instance_type" {
  description = "MSK broker instance type"
  type        = string
  default     = "kafka.m5.large"
}

variable "kafka_ebs_volume_gb" {
  description = "EBS volume size per Kafka broker in GB"
  type        = number
  default     = 500
}

# ---------------------------------------------------------------------------
# Disaster Recovery
# ---------------------------------------------------------------------------

variable "enable_dr" {
  description = "Enable disaster recovery (cross-region replication)"
  type        = bool
  default     = false
}

variable "dr_region" {
  description = "DR region for cross-region replication"
  type        = string
  default     = "us-west-2"
}
