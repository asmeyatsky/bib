# Bank-in-a-Box Infrastructure
# Main Terraform configuration for provisioning cloud resources.
# Default provider is AWS; set var.cloud_provider to switch.

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket         = "bib-terraform-state"
    key            = "infrastructure/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "bib-terraform-locks"
  }
}

# ---------------------------------------------------------------------------
# Provider Configuration
# ---------------------------------------------------------------------------

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "bank-in-a-box"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

# ---------------------------------------------------------------------------
# Data Sources
# ---------------------------------------------------------------------------

data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_caller_identity" "current" {}

# ---------------------------------------------------------------------------
# VPC
# ---------------------------------------------------------------------------

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "${var.project_name}-${var.environment}-vpc"
  }
}

resource "aws_subnet" "private" {
  count             = min(length(data.aws_availability_zones.available.names), 3)
  vpc_id            = aws_vpc.main.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 4, count.index)
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name = "${var.project_name}-${var.environment}-private-${count.index}"
    Tier = "private"
  }
}

resource "aws_subnet" "public" {
  count                   = min(length(data.aws_availability_zones.available.names), 3)
  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 4, count.index + 8)
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.project_name}-${var.environment}-public-${count.index}"
    Tier = "public"
  }
}

# ---------------------------------------------------------------------------
# Modules
# ---------------------------------------------------------------------------

module "kubernetes" {
  source = "./modules/kubernetes"

  project_name    = var.project_name
  environment     = var.environment
  cloud_provider  = var.cloud_provider
  vpc_id          = aws_vpc.main.id
  subnet_ids      = aws_subnet.private[*].id
  cluster_version = var.kubernetes_version
  node_min_count  = var.node_min_count
  node_max_count  = var.node_max_count
  node_instance_type = var.node_instance_type
}

module "database" {
  source = "./modules/database"

  project_name    = var.project_name
  environment     = var.environment
  cloud_provider  = var.cloud_provider
  vpc_id          = aws_vpc.main.id
  subnet_ids      = aws_subnet.private[*].id
  db_engine       = var.db_engine
  db_version      = var.db_version
  db_instance_class = var.db_instance_class
  db_storage_gb   = var.db_storage_gb
  multi_az        = var.db_multi_az
  backup_retention_days = var.db_backup_retention_days
  enable_replication    = var.enable_dr
  replica_region        = var.dr_region
}

module "kafka" {
  source = "./modules/kafka"

  project_name   = var.project_name
  environment    = var.environment
  cloud_provider = var.cloud_provider
  vpc_id         = aws_vpc.main.id
  subnet_ids     = aws_subnet.private[*].id
  broker_count   = var.kafka_broker_count
  instance_type  = var.kafka_instance_type
  ebs_volume_gb  = var.kafka_ebs_volume_gb
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "vpc_id" {
  value = aws_vpc.main.id
}

output "kubernetes_cluster_endpoint" {
  value = module.kubernetes.cluster_endpoint
}

output "database_endpoint" {
  value     = module.database.endpoint
  sensitive = true
}

output "kafka_bootstrap_brokers" {
  value = module.kafka.bootstrap_brokers
}
