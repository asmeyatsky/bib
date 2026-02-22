# Database Module
# Provisions RDS PostgreSQL with encryption, automated backups, and
# optional cross-region replication for disaster recovery.

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

variable "db_engine" {
  type    = string
  default = "postgres"
}

variable "db_version" {
  type    = string
  default = "16.2"
}

variable "db_instance_class" {
  type    = string
  default = "db.r6g.xlarge"
}

variable "db_storage_gb" {
  type    = number
  default = 100
}

variable "multi_az" {
  type    = bool
  default = true
}

variable "backup_retention_days" {
  type    = number
  default = 30
}

variable "enable_replication" {
  type    = bool
  default = false
}

variable "replica_region" {
  type    = string
  default = "us-west-2"
}

locals {
  db_identifier = "${var.project_name}-${var.environment}"
  is_aws        = var.cloud_provider == "aws"
}

# ---------------------------------------------------------------------------
# Subnet Group
# ---------------------------------------------------------------------------

resource "aws_db_subnet_group" "main" {
  count      = local.is_aws ? 1 : 0
  name       = "${local.db_identifier}-subnet-group"
  subnet_ids = var.subnet_ids

  tags = {
    Name = "${local.db_identifier}-subnet-group"
  }
}

# ---------------------------------------------------------------------------
# Security Group
# ---------------------------------------------------------------------------

resource "aws_security_group" "database" {
  count       = local.is_aws ? 1 : 0
  name        = "${local.db_identifier}-db-sg"
  description = "Security group for ${local.db_identifier} RDS"
  vpc_id      = var.vpc_id

  ingress {
    description = "PostgreSQL from VPC"
    from_port   = 5432
    to_port     = 5432
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
    Name = "${local.db_identifier}-db-sg"
  }
}

data "aws_vpc" "main" {
  count = local.is_aws ? 1 : 0
  id    = var.vpc_id
}

# ---------------------------------------------------------------------------
# KMS Encryption Key
# ---------------------------------------------------------------------------

resource "aws_kms_key" "database" {
  count                   = local.is_aws ? 1 : 0
  description             = "KMS key for RDS encryption - ${local.db_identifier}"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = {
    Name = "${local.db_identifier}-rds-kms"
  }
}

# ---------------------------------------------------------------------------
# RDS Parameter Group
# ---------------------------------------------------------------------------

resource "aws_db_parameter_group" "main" {
  count  = local.is_aws ? 1 : 0
  name   = "${local.db_identifier}-pg"
  family = "postgres16"

  parameter {
    name  = "log_connections"
    value = "1"
  }

  parameter {
    name  = "log_disconnections"
    value = "1"
  }

  parameter {
    name  = "log_statement"
    value = "ddl"
  }

  parameter {
    name         = "rds.force_ssl"
    value        = "1"
    apply_method = "pending-reboot"
  }

  tags = {
    Name = "${local.db_identifier}-pg"
  }
}

# ---------------------------------------------------------------------------
# Primary RDS Instance
# ---------------------------------------------------------------------------

resource "aws_db_instance" "primary" {
  count = local.is_aws ? 1 : 0

  identifier     = "${local.db_identifier}-primary"
  engine         = var.db_engine
  engine_version = var.db_version
  instance_class = var.db_instance_class

  allocated_storage     = var.db_storage_gb
  max_allocated_storage = var.db_storage_gb * 2
  storage_type          = "gp3"
  storage_encrypted     = true
  kms_key_id            = aws_kms_key.database[0].arn

  db_subnet_group_name   = aws_db_subnet_group.main[0].name
  vpc_security_group_ids = [aws_security_group.database[0].id]
  parameter_group_name   = aws_db_parameter_group.main[0].name

  multi_az            = var.multi_az
  publicly_accessible = false

  backup_retention_period = var.backup_retention_days
  backup_window           = "03:00-04:00"
  maintenance_window      = "sun:04:00-sun:05:00"

  deletion_protection       = var.environment == "prod"
  skip_final_snapshot       = var.environment != "prod"
  final_snapshot_identifier = var.environment == "prod" ? "${local.db_identifier}-final" : null

  performance_insights_enabled = true
  monitoring_interval          = 60

  tags = {
    Name        = "${local.db_identifier}-primary"
    Environment = var.environment
  }
}

# ---------------------------------------------------------------------------
# Cross-Region Read Replica (DR)
# ---------------------------------------------------------------------------

resource "aws_db_instance" "replica" {
  count = local.is_aws && var.enable_replication ? 1 : 0

  identifier          = "${local.db_identifier}-replica"
  replicate_source_db = aws_db_instance.primary[0].arn
  instance_class      = var.db_instance_class

  storage_encrypted = true
  kms_key_id        = aws_kms_key.database[0].arn

  publicly_accessible = false

  performance_insights_enabled = true
  monitoring_interval          = 60

  tags = {
    Name        = "${local.db_identifier}-replica"
    Environment = var.environment
    Role        = "dr-replica"
  }
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "endpoint" {
  value = local.is_aws ? aws_db_instance.primary[0].endpoint : "placeholder"
}

output "replica_endpoint" {
  value = local.is_aws && var.enable_replication ? aws_db_instance.replica[0].endpoint : ""
}

output "security_group_id" {
  value = local.is_aws ? aws_security_group.database[0].id : ""
}
