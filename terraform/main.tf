terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Data source for latest Amazon Linux 2023 AMI
data "aws_ami" "amazon_linux" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# VPC
resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(var.tags, {
    Name = "task-queue-vpc"
  })
}

# Internet Gateway
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = merge(var.tags, {
    Name = "task-queue-igw"
  })
}

# Public Subnet
resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.public_subnet_cidr
  availability_zone       = data.aws_availability_zones.available.names[0]
  map_public_ip_on_launch = true

  tags = merge(var.tags, {
    Name = "task-queue-public-subnet"
  })
}

data "aws_availability_zones" "available" {
  state = "available"
}

# Route Table for Public Subnet
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = merge(var.tags, {
    Name = "task-queue-public-rt"
  })
}

resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

# Security Groups
resource "aws_security_group" "redis" {
  name        = "task-queue-redis-sg"
  description = "Security group for Redis instance"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "Redis from API"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.api.id, aws_security_group.worker.id]
  }

  egress {
    description = "Allow all outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, {
    Name = "task-queue-redis-sg"
  })
}

resource "aws_security_group" "api" {
  name        = "task-queue-api-sg"
  description = "Security group for API service"
  vpc_id      = aws_vpc.main.id

  ingress {
    description = "HTTP API from anywhere (for laptop access)"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = [var.allowed_cidr]
  }

  egress {
    description = "Allow all outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, {
    Name = "task-queue-api-sg"
  })
}

resource "aws_security_group" "worker" {
  name        = "task-queue-worker-sg"
  description = "Security group for worker instances"
  vpc_id      = aws_vpc.main.id

  egress {
    description = "Allow all outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, {
    Name = "task-queue-worker-sg"
  })
}

# Redis EC2 Instance
resource "aws_instance" "redis" {
  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = "t3.micro"
  key_name               = var.key_name
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.redis.id]

  root_block_device {
    volume_size = 30
    volume_type = "gp3"
  }

  user_data = templatefile("${path.module}/user_data/redis.sh", {
    redis_image = var.redis_image
  })

  tags = merge(var.tags, {
    Name = "task-queue-redis"
  })
}

# API EC2 Instance
resource "aws_instance" "api" {
  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = "t3.small"
  key_name               = var.key_name
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.api.id]

  root_block_device {
    volume_size = 30
    volume_type = "gp3"
  }

  user_data = templatefile("${path.module}/user_data/api.sh", {
    api_image        = var.api_image
    redis_private_ip = aws_instance.redis.private_ip
  })

  tags = merge(var.tags, {
    Name = "task-queue-api"
  })

  depends_on = [aws_instance.redis]
}

# Worker Launch Template
resource "aws_launch_template" "worker" {
  name_prefix   = "task-queue-worker-"
  image_id      = data.aws_ami.amazon_linux.id
  instance_type = "t3.micro"
  key_name      = var.key_name

  vpc_security_group_ids = [aws_security_group.worker.id]

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 30
      volume_type = "gp3"
    }
  }

  user_data = base64encode(templatefile("${path.module}/user_data/worker.sh", {
    worker_image    = var.worker_image
    redis_private_ip = aws_instance.redis.private_ip
    queue_type      = var.worker_queue_type
    mode            = var.worker_mode
  }))

  tag_specifications {
    resource_type = "instance"
    tags = merge(var.tags, {
      Name = "task-queue-worker"
    })
  }
}

# Worker Auto Scaling Group
resource "aws_autoscaling_group" "worker" {
  name                = "task-queue-worker-asg"
  vpc_zone_identifier = [aws_subnet.public.id]
  target_group_arns   = [] # No load balancer needed
  health_check_type   = "EC2"
  health_check_grace_period = 300

  min_size         = var.worker_min_size
  max_size         = var.worker_max_size
  desired_capacity = var.worker_desired_capacity

  launch_template {
    id      = aws_launch_template.worker.id
    version = "$Latest"
  }

  tag {
    key                 = "Name"
    value               = "task-queue-worker"
    propagate_at_launch = true
  }

  dynamic "tag" {
    for_each = var.tags
    content {
      key                 = tag.key
      value               = tag.value
      propagate_at_launch = true
    }
  }
}

