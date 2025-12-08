# Deployment Guide

## Prerequisites

1. **AWS Account** with appropriate permissions
2. **AWS CLI** configured (`aws configure`)
3. **Terraform** >= 1.0 installed
4. **Docker images** built and pushed to ECR or Docker Hub
5. **AWS Key Pair** created in your region

## Step 1: Prepare Docker Images

### Build and Push API Image

```bash
cd src

# Build API image
docker build -f api/main/Dockerfile -t your-ecr-repo/api:latest .

# Tag for ECR (replace with your account ID and region)
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com
docker tag your-ecr-repo/api:latest 123456789012.dkr.ecr.us-east-1.amazonaws.com/api:latest
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/api:latest
```

### Build and Push Worker Image

```bash
# Build worker image
docker build -f worker/Dockerfile -t your-ecr-repo/worker:latest .

# Tag and push
docker tag your-ecr-repo/worker:latest 123456789012.dkr.ecr.us-east-1.amazonaws.com/worker:latest
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/worker:latest
```

### Alternative: Use Docker Hub

If using Docker Hub instead of ECR:

```bash
docker tag your-ecr-repo/api:latest your-dockerhub-username/api:latest
docker push your-dockerhub-username/api:latest
```

## Step 2: Configure Terraform

```bash
cd terraform

# Copy example variables file
cp terraform.tfvars.example terraform.tfvars

# Edit terraform.tfvars with your values
# - key_name: Your AWS key pair name
# - api_image: Your ECR/Docker Hub image URL
# - worker_image: Your ECR/Docker Hub image URL
```

## Step 3: Deploy Infrastructure

```bash
# Initialize Terraform
terraform init

# Review the plan
terraform plan

# Apply the configuration
terraform apply

# Type 'yes' when prompted
```

## Step 4: Get API Endpoint

After deployment completes:

```bash
# Get API endpoint
terraform output api_endpoint

# Or get all outputs
terraform output
```

The output will show:
```
api_endpoint = "http://54.123.45.67:8080"
```

## Step 5: Update Client Code (Optional)

You can either:

### Option A: Use Environment Variable

Modify client code to read from environment:

```go
baseURL := os.Getenv("API_ENDPOINT")
if baseURL == "" {
    baseURL = "http://localhost:8080"
}
```

Then run:
```bash
export API_ENDPOINT="http://<api-public-ip>:8080"
go run ./client/exp1/exp1_loadtest.go
```

### Option B: Hardcode for Testing

Temporarily modify client files:

```go
const baseURL = "http://<api-public-ip>:8080"
```

## Step 6: Run Experiments

### Experiment 1: Task Length Distribution

```bash
cd src
export API_ENDPOINT="http://$(cd ../terraform && terraform output -raw api_public_ip):8080"
go run ./client/exp1/exp1_loadtest.go
```

### Experiment 2: Worker Scaling

**Test with 1 worker:**
```bash
# Workers start with desired_capacity=1 by default
cd terraform
terraform apply -var="worker_desired_capacity=1"
cd ../src
go run ./client/exp2/exp2_loadtest.go
```

**Test with 5 workers:**
```bash
cd terraform
terraform apply -var="worker_desired_capacity=5"
# Wait ~2 minutes for instances to launch
cd ../src
go run ./client/exp2/exp2_loadtest.go
```

**Or use AWS CLI:**
```bash
aws autoscaling set-desired-capacity \
  --auto-scaling-group-name $(terraform output -raw worker_asg_name) \
  --desired-capacity 5 \
  --region us-east-1
```

### Experiment 3: Retry Mechanism

```bash
# Update worker mode to retry
cd terraform
terraform apply -var="worker_mode=retry"
cd ../src
go run ./client/exp3/exp3_loadtest.go
```

## Step 7: Monitor and Debug

### Check API Status

```bash
API_IP=$(cd terraform && terraform output -raw api_public_ip)
curl http://$API_IP:8080/queue/status
```

### Check Worker Instances

```bash
aws ec2 describe-instances \
  --filters "Name=tag:Name,Values=task-queue-worker" \
  --query 'Reservations[*].Instances[*].[InstanceId,State.Name,PrivateIpAddress]' \
  --output table
```

### SSH to Instances

```bash
# Get instance IPs
API_IP=$(cd terraform && terraform output -raw api_public_ip)
REDIS_IP=$(aws ec2 describe-instances \
  --filters "Name=tag:Name,Values=task-queue-redis" \
  --query 'Reservations[0].Instances[0].PublicIpAddress' \
  --output text)

# SSH to API
ssh -i ~/.ssh/your-key.pem ec2-user@$API_IP

# Check API logs
sudo docker logs api

# SSH to Redis
ssh -i ~/.ssh/your-key.pem ec2-user@$REDIS_IP

# Check Redis
sudo docker exec -it redis redis-cli ping
sudo docker exec -it redis redis-cli LLEN task:queue
```

## Step 8: Cleanup

```bash
cd terraform
terraform destroy

# Type 'yes' when prompted
```

## Troubleshooting

### API not accessible

1. Check security group allows port 8080 from your IP
2. Check API instance is running: `aws ec2 describe-instances --filters "Name=tag:Name,Values=task-queue-api"`
3. Check API logs: SSH to instance and run `sudo docker logs api`

### Workers not processing tasks

1. Check workers can reach Redis (same VPC, security group allows 6379)
2. Check worker logs: SSH to worker instance and run `sudo docker logs worker`
3. Verify REDIS_ADDR environment variable

### Docker images not found

1. Verify images exist in ECR/Docker Hub
2. Check IAM role has ECR permissions (if using ECR)
3. Check user_data script logs: `/var/log/user-data.log`

## Cost Estimation

Approximate monthly costs (us-east-1):

- Redis (t3.micro): ~$7.50/month
- API (t3.small): ~$15/month
- Workers (t3.micro x 1-10): ~$7.50-$75/month
- Data transfer: Minimal for experiments

**Total**: ~$30-100/month depending on worker count

