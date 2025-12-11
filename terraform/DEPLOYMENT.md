# Distributed Task Queue - Cloud Deployment Guide

## 1. Overview

**Cloud Architecture (Minimal Viable Version):**

- **VPC + Subnets**
- **Redis EC2**: Private IP, no public exposure
- **API EC2**: Public IP, exposes port 8080
  - User data script installs Docker, pulls `luluxxu/dtq-api:latest`
  - Runs API with `REDIS_ADDR=<redis_private_ip>:6379`
- **Worker Auto Scaling Group (ASG)**
  - One or more EC2 instances, each runs `luluxxu/dtq-worker:latest`
  - Workers connect to Redis and consume tasks from the queue
- **Your Local Machine**
  - Runs the Go client (exp1/exp2/exp3), pointing to the cloud API endpoint

## 2. Prerequisites

**On your local machine:**

1. **AWS Account** with permissions to create:
   - EC2 instances
   - VPC, subnets, security groups
   - Auto Scaling Group

2. **AWS CLI** installed and configured:
```bash
   aws configure
   # Make sure region matches your Terraform config, e.g. us-west-2
```

3. **Terraform** ≥ 1.0 installed:
```bash
   terraform version
```

4. **Docker** installed

5. **AWS EC2 Key Pair** created in the same region (e.g. us-west-2):
   - In AWS Console → EC2 → Key Pairs → Create key pair
   - Download `taskqueue-key.pem` to local, like `~/.ssh/taskqueue-key.pem`
   - Set permissions: `chmod 400 ~/.ssh/taskqueue-key.pem`
   - In `terraform.tfvars`: `key_name = "taskqueue-key"`

## 3. Build & Push Docker Images (Docker Hub)

**One-time setup:** Build images on your local machine, then push to Docker Hub public repository so EC2 instances in the cloud can pull them.

From the repo root:
```bash
cd src
```

### 3.1 API Image
```bash
# Build for linux/amd64 (EC2 uses x86_64 architecture)
docker build \
  --platform linux/amd64 \
  -f api/main/Dockerfile \
  -t luluxxu/dtq-api:latest \
  .

# Push to Docker Hub
docker push luluxxu/dtq-api:latest
```

### 3.2 Worker Image
```bash
docker build \
  --platform linux/amd64 \
  -f worker/Dockerfile \
  -t luluxxu/dtq-worker:latest \
  .

docker push luluxxu/dtq-worker:latest
```

> **Note:** If you're not using `luluxxu/...`, change it to your own Docker Hub username and update `api_image` / `worker_image` in `terraform.tfvars` accordingly.

## 4. Configure Terraform Variables
```bash
cd ../terraform
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` (key fields example):
```hcl
aws_region = "us-west-2"

# EC2 Key Pair name (must exist in AWS)
key_name = "taskqueue-key"

# Public Docker Hub images (already pushed)
api_image    = "luluxxu/dtq-api:latest"
worker_image = "luluxxu/dtq-worker:latest"
redis_image  = "redis:7-alpine"

# Worker scaling (for Experiment 2)
worker_min_size         = 1
worker_max_size         = 10
worker_desired_capacity = 1

# Worker behavior
worker_queue_type = "fifo"   # or "priority"
worker_mode       = "simple" # or "retry"

# (Optional) Restrict API access to your IP
# allowed_cidr = "YOUR_IP/32"
```

## 5. Deploy with Terraform

Execute in the `terraform/` directory:
```bash
# Initialize Terraform
terraform init

# Review the plan
terraform plan

# Actually create resources
terraform apply
# When prompted "Enter a value for var.key_name", enter the key_name 
# you configured in tfvars (e.g., taskqueue-key)
# Enter yes to confirm
```

After success, you'll see output similar to:
```
Apply complete! Resources: X added, Y changed, Z destroyed.

Outputs:

api_endpoint     = "http://52.41.50.47:8080"
api_public_ip    = "52.41.50.47"
redis_private_ip = "10.0.1.8"
worker_asg_name  = "task-queue-worker-asg"
...
```

**Note down:**
- `api_endpoint` / `api_public_ip`
- `redis_private_ip`
- `worker_asg_name`

## 6. Smoke Test: Check API & Queue

On your local machine:
```bash
cd terraform

# Get API Endpoint
terraform output api_endpoint
# Or just the IP
terraform output -raw api_public_ip
```

**Quick test:**
```bash
API_IP=$(terraform output -raw api_public_ip)
curl http://$API_IP:8080/queue/status
```

**Expected output:**
```json
{"fifo_queue_length":0,"priority_queue_length":0,"total_backlog":0}
```

If JSON is returned, it means:
- Docker has started the API container on EC2
- API can connect to Redis

## 7. Run Experiments from Your Laptop

### 7.1 Common Setup

In your local terminal:
```bash
cd ../src
export API_ENDPOINT="http://$(cd ../terraform && terraform output -raw api_public_ip):8080"
echo $API_ENDPOINT
# Confirm it's something like http://52.41.50.47:8080
```

> **Note:** All client programs will prioritize using `API_ENDPOINT`, otherwise default to `http://localhost:8080`.

### 7.2 Experiment 1 – Task Length Distribution
```bash
cd src
export API_ENDPOINT="http://$(cd ../terraform && terraform output -raw api_public_ip):8080"

go run ./client/exp1/exp1_loadtest.go
```

**You should see logs similar to:**
```
Using API endpoint: http://52.41.50.47:8080
Submitted 0 to http://52.41.50.47:8080/task/fifo status: 201 Created
Submitted 1 to http://52.41.50.47:8080/task/pq status: 201 Created
...
```

**Check cloud queue status:**
```bash
curl http://$API_ENDPOINT/queue/status
# {"fifo_queue_length":..., "priority_queue_length":..., "total_backlog":...}
```

**Worker logs (optional, SSH into worker instance to view):**
```bash
ssh -i ~/.ssh/taskqueue-key.pem ec2-user@<worker_public_ip>
sudo docker logs -f dtq-worker
```

### 7.3 Experiment 2 – Worker Scaling & Backlog Clearance

Set different worker counts to test how long it takes to clear 5000 tasks (the exp2 client will test this for you).

**Scale to 1 worker:**
```bash
cd terraform
aws autoscaling set-desired-capacity \
  --auto-scaling-group-name $(terraform output -raw worker_asg_name) \
  --desired-capacity 1 \
  --region us-west-2
```

Wait 1–2 minutes for EC2 to start.

#### Step 1: Verify API is Running
```bash
cd terraform
API_IP=$(terraform output -raw api_public_ip)
curl http://$API_IP:8080/queue/status
```

**Should return:**
```json
{"fifo_queue_length":0,"priority_queue_length":0,"total_backlog":0}
```

#### Step 2: Set Environment Variable and Run Experiment
```bash
cd src

# Set API endpoint
export API_ENDPOINT="http://$(cd ../terraform && terraform output -raw api_public_ip):8080"
echo "Using API endpoint: $API_ENDPOINT"

# Run Experiment 2
go run ./client/exp2/exp2_loadtest.go
```

#### Step 3: Monitor Queue Status (in another terminal)
```bash
cd terraform
API_IP=$(terraform output -raw api_public_ip)

while true; do  
  clear  
  echo "=== $(date '+%H:%M:%S') ==="  
  curl -s http://$API_IP:8080/queue/status | python3 -m json.tool 2>/dev/null || curl -s http://$API_IP:8080/queue/status  
  echo ""  
  sleep 5
done
```

#### Step 4: Monitor Worker Logs (in worker SSH session)
```bash
# SSH into worker instance first
ssh -i ~/.ssh/taskqueue-key.pem ec2-user@<worker_public_ip>

# Then monitor logs
sudo docker logs -f dtq-worker
```

### 7.4 Experiment 3 – Retry & Failure Injection

To switch to retry mode, there are two ways:

#### A. Use Terraform Variables (Recommended for Reports)

Edit `terraform.tfvars`:
```hcl
worker_mode = "retry"
```

Then apply:
```bash
cd terraform
terraform apply
# This will make new worker instances start with `-mode=retry`
```

Execute on local machine:
```bash
cd ../src
export API_ENDPOINT="http://$(cd ../terraform && terraform output -raw api_public_ip):8080"
go run ./client/exp3/exp3_loadtest.go
```

#### B. Temporarily Adjust Manually on a Worker EC2 (For Debugging)

SSH into worker EC2 and manually use:
```bash
sudo docker rm -f dtq-worker 2>/dev/null || true
sudo docker run -d --restart always \
  --name dtq-worker \
  -e REDIS_ADDR="10.0.1.8:6379" \
  luluxxu/dtq-worker:latest \
  ./worker -queue=fifo -mode=retry
```

## 8. SSH into API / Worker (Optional)

### API Instance
```bash
cd terraform
API_IP=$(terraform output -raw api_public_ip)

ssh -i ~/.ssh/taskqueue-key.pem ec2-user@$API_IP

# View containers
sudo docker ps
sudo docker logs dtq-api --tail 50
```

### Worker Instances
```bash
# List all worker instances
aws ec2 describe-instances \
  --filters "Name=tag:Name,Values=task-queue-worker" \
  --region us-west-2 \
  --query 'Reservations[*].Instances[*].[InstanceId,State.Name,PrivateIpAddress,PublicIpAddress]' \
  --output table

# SSH into a specific worker (use PublicIp from above)
ssh -i ~/.ssh/taskqueue-key.pem ec2-user@<worker_public_ip>

# View containers
sudo docker ps
sudo docker logs dtq-worker --tail 50
```

## 9. Tear Down

After experiments, don't forget to destroy cloud resources to avoid charges:
```bash
cd terraform
terraform destroy
# Enter yes to confirm
```

## 10. Tips / Troubleshooting

### Issue: curl 8080 connection fails

**Solutions:**
- Check if Terraform apply was successful
- Check if `terraform output api_public_ip` matches the IP you're curling
- SSH into API instance and check:
```bash
  sudo docker ps  # to see if dtq-api exists
  sudo ss -tulpn | grep 8080
```

### Issue: API container won't start

**Most likely causes:**
- Image architecture issue
- `REDIS_ADDR` configuration issue

**Solutions:**
- Ensure images are built with `--platform linux/amd64`
- Ensure user_data uses `-e REDIS_ADDR="${redis_private_ip}:6379"`

### Issue: Tasks are never consumed

**Check if worker is running:**
```bash
sudo docker ps  # to see if dtq-worker exists
sudo docker logs dtq-worker  # to see if it shows "Processing task ..."
```

**Verify configuration:**
- Confirm worker environment variable `REDIS_ADDR` points to the correct `redis_private_ip:6379`