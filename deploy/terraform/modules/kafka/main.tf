# Kafka Module
# Provisions Amazon MSK (Managed Streaming for Apache Kafka) with
# encryption and monitoring. Supports AWS MSK or Confluent Cloud.

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

variable "broker_count" {
  type    = number
  default = 3
}

variable "instance_type" {
  type    = string
  default = "kafka.m5.large"
}

variable "ebs_volume_gb" {
  type    = number
  default = 500
}

locals {
  cluster_name = "${var.project_name}-${var.environment}-kafka"
  is_aws       = var.cloud_provider == "aws"
}

# ---------------------------------------------------------------------------
# Security Group
# ---------------------------------------------------------------------------

resource "aws_security_group" "kafka" {
  count       = local.is_aws ? 1 : 0
  name        = "${local.cluster_name}-sg"
  description = "Security group for ${local.cluster_name} MSK"
  vpc_id      = var.vpc_id

  ingress {
    description = "Kafka TLS from VPC"
    from_port   = 9094
    to_port     = 9094
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.main[0].cidr_block]
  }

  ingress {
    description = "Kafka plaintext from VPC"
    from_port   = 9092
    to_port     = 9092
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.main[0].cidr_block]
  }

  ingress {
    description = "ZooKeeper from VPC"
    from_port   = 2181
    to_port     = 2181
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.main[0].cidr_block]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${local.cluster_name}-sg"
  }
}

data "aws_vpc" "main" {
  count = local.is_aws ? 1 : 0
  id    = var.vpc_id
}

# ---------------------------------------------------------------------------
# KMS Key for Encryption
# ---------------------------------------------------------------------------

resource "aws_kms_key" "kafka" {
  count                   = local.is_aws ? 1 : 0
  description             = "KMS key for MSK encryption - ${local.cluster_name}"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = {
    Name = "${local.cluster_name}-kms"
  }
}

# ---------------------------------------------------------------------------
# MSK Configuration
# ---------------------------------------------------------------------------

resource "aws_msk_configuration" "main" {
  count          = local.is_aws ? 1 : 0
  name           = "${local.cluster_name}-config"
  kafka_versions = ["3.6.0"]

  server_properties = <<PROPERTIES
auto.create.topics.enable=false
default.replication.factor=3
min.insync.replicas=2
num.partitions=6
log.retention.hours=168
log.retention.bytes=1073741824
unclean.leader.election.enable=false
PROPERTIES
}

# ---------------------------------------------------------------------------
# MSK Cluster
# ---------------------------------------------------------------------------

resource "aws_msk_cluster" "main" {
  count         = local.is_aws ? 1 : 0
  cluster_name  = local.cluster_name
  kafka_version = "3.6.0"

  number_of_broker_nodes = var.broker_count

  broker_node_group_info {
    instance_type  = var.instance_type
    client_subnets = var.subnet_ids

    storage_info {
      ebs_storage_info {
        volume_size = var.ebs_volume_gb
      }
    }

    security_groups = [aws_security_group.kafka[0].id]
  }

  configuration_info {
    arn      = aws_msk_configuration.main[0].arn
    revision = aws_msk_configuration.main[0].latest_revision
  }

  encryption_info {
    encryption_at_rest_kms_key_arn = aws_kms_key.kafka[0].arn

    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
  }

  open_monitoring {
    prometheus {
      jmx_exporter {
        enabled_in_broker = true
      }
      node_exporter {
        enabled_in_broker = true
      }
    }
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.kafka[0].name
      }
    }
  }

  tags = {
    Name        = local.cluster_name
    Environment = var.environment
  }
}

# ---------------------------------------------------------------------------
# CloudWatch Log Group
# ---------------------------------------------------------------------------

resource "aws_cloudwatch_log_group" "kafka" {
  count             = local.is_aws ? 1 : 0
  name              = "/msk/${local.cluster_name}"
  retention_in_days = 30

  tags = {
    Name = "${local.cluster_name}-logs"
  }
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "bootstrap_brokers" {
  value = local.is_aws ? aws_msk_cluster.main[0].bootstrap_brokers_tls : ""
}

output "cluster_arn" {
  value = local.is_aws ? aws_msk_cluster.main[0].arn : ""
}

output "zookeeper_connect" {
  value = local.is_aws ? aws_msk_cluster.main[0].zookeeper_connect_string : ""
}
