# Terraform Deployment for Distributed Task Queue

## Summary

- **System**: Go-based distributed task queue with HTTP API, Redis queues, and worker pool
- **Components**: API service (port 8080), Redis (port 6379), scalable workers
- **Deployment**: AWS EC2 instances with Docker, VPC with public subnet, Auto Scaling Group for workers
- **Experiments**: Supports all 3 experiments with API accessible from laptop, scalable worker pool for Experiment 2

## Architecture Design

```
┌─────────────────────────────────────────────────────────┐
│                         VPC                               │
│  ┌──────────────────────────────────────────────────┐   │
│  │            Public Subnet (10.0.1.0/24)          │   │
│  │                                                  │   │
│  │  ┌──────────────┐    ┌──────────────┐          │   │
│  │  │   Redis EC2  │    │   API EC2    │          │   │
│  │  │  (port 6379) │    │  (port 8080) │          │   │
│  │  └──────┬───────┘    └──────┬───────┘          │   │
│  │         │                   │                   │   │
│  │         └─────────┬─────────┘                   │   │
│  │                   │                             │   │
│  │         ┌─────────▼─────────┐                   │   │
│  │         │  Worker ASG       │                   │   │
│  │         │  (1-10 instances)  │                   │   │
│  │         └───────────────────┘                   │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
         ▲
         │ HTTP (port 8080)
         │
    ┌────┴────┐
    │ Laptop  │
    │ (Client)│
    └─────────┘
```

### Components

1. **VPC & Networking**
   - VPC with public subnet
   - Internet Gateway for public access
   - Route tables

2. **Redis Instance**
   - Single EC2 instance (t3.micro or t3.small)
   - Security group: Allow port 6379 from API and Workers
   - User data: Install Docker, run Redis container

3. **API Instance**
   - Single EC2 instance (t3.small)
   - Security group: Allow port 8080 from anywhere (0.0.0.0/0)
   - User data: Install Docker, run API container
   - Environment: REDIS_ADDR=<redis-private-ip>:6379

4. **Worker Auto Scaling Group**
   - ASG with min=1, max=10, desired=1
   - Launch template with user data for Docker + worker container
   - Environment: REDIS_ADDR=<redis-private-ip>:6379
   - Supports scaling for Experiment 2

5. **Security Groups**
   - `redis_sg`: Allow 6379 from API and Worker security groups
   - `api_sg`: Allow 8080 from 0.0.0.0/0 (for laptop access)
   - `worker_sg`: Allow outbound to Redis

## File Layout

```
terraform/
├── main.tf                 # Main resources (VPC, EC2, ASG, SGs)
├── variables.tf            # Input variables
├── outputs.tf              # Output values (API endpoint, Redis IP)
├── user_data/
│   ├── redis.sh           # Redis instance startup script
│   ├── api.sh             # API instance startup script
│   └── worker.sh          # Worker instance startup script
└── README.md              # This file
```

## Assumptions

- **Region**: us-east-1 (configurable via variable)
- **AMI**: Latest Amazon Linux 2023 AMI (auto-discovered)
- **Instance Types**: 
  - Redis: t3.micro
  - API: t3.small
  - Workers: t3.micro
- **Key Pair**: Must be created manually and provided via variable
- **Docker Images**: Assumed to be in ECR or Docker Hub (configurable)
- **No Redis Password**: Matches current local setup

