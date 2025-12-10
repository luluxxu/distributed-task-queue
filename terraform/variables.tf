variable "aws_region" {
  description = "AWS region for deployment"
  type        = string
  default     = "us-west-2"
}

variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "public_subnet_cidr" {
  description = "CIDR block for public subnet"
  type        = string
  default     = "10.0.1.0/24"
}

variable "key_name" {
  description = "Name of AWS key pair for EC2 instances"
  type        = string
}

variable "api_image" {
  description = "Docker image for API service"
  type        = string
  default     = "your-ecr-repo/api:latest"
}

variable "worker_image" {
  description = "Docker image for worker service"
  type        = string
  default     = "your-ecr-repo/worker:latest"
}

variable "redis_image" {
  description = "Docker image for Redis"
  type        = string
  default     = "redis:7-alpine"
}

variable "worker_min_size" {
  description = "Minimum number of worker instances"
  type        = number
  default     = 10
}

variable "worker_max_size" {
  description = "Maximum number of worker instances"
  type        = number
  default     = 10
}

variable "worker_desired_capacity" {
  description = "Desired number of worker instances (for Experiment 2)"
  type        = number
  default     = 10
}

variable "worker_queue_type" {
  description = "Queue type for workers (fifo or priority)"
  type        = string
  default     = "priority"
}

variable "worker_mode" {
  description = "Worker mode (simple or retry)"
  type        = string
  default     = "simple"
}

variable "allowed_cidr" {
  description = "CIDR block allowed to access API (your laptop IP)"
  type        = string
  default     = "0.0.0.0/0"  # Change to your IP for better security
}

variable "tags" {
  description = "Common tags for all resources"
  type        = map(string)
  default = {
    Project     = "distributed-task-queue"
    Environment = "experiment"
  }
}

