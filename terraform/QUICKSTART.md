# Quick Start Guide

## Prerequisites Check

```bash
# Check AWS CLI
aws --version

# Check Terraform
terraform version

# Check AWS credentials
aws sts get-caller-identity
```

## 5-Minute Deployment

### 1. Configure Variables

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars

# Edit terraform.tfvars - minimum required:
# key_name = "your-existing-key-pair-name"
```

### 2. Deploy

```bash
terraform init
terraform plan
terraform apply  # Type 'yes'
```

### 3. Get API Endpoint

```bash
terraform output api_endpoint
# Example: http://54.123.45.67:8080
```

### 4. Test API

```bash
API_IP=$(terraform output -raw api_public_ip)
curl http://$API_IP:8080/queue/status
```

### 5. Run Experiment (from your laptop)

```bash
cd ../src

# Update client code temporarily or use environment variable
# For exp1_loadtest.go, change:
# urlFIFO := "http://$API_IP:8080/task/fifo"
# urlPQ := "http://$API_IP:8080/task/pq"

go run ./client/exp1/exp1_loadtest.go
```

## Scale Workers for Experiment 2

```bash
# Scale to 5 workers
aws autoscaling set-desired-capacity \
  --auto-scaling-group-name $(terraform output -raw worker_asg_name) \
  --desired-capacity 5 \
  --region $(terraform output -raw aws_region)

# Wait 2-3 minutes for instances to launch, then run:
cd src
go run ./client/exp2/exp2_loadtest.go
```

## Cleanup

```bash
terraform destroy
```

