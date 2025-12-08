output "api_public_ip" {
  description = "Public IP address of the API instance"
  value       = aws_instance.api.public_ip
}

output "api_public_dns" {
  description = "Public DNS name of the API instance"
  value       = aws_instance.api.public_dns
}

output "api_endpoint" {
  description = "API endpoint URL for client load tests"
  value       = "http://${aws_instance.api.public_ip}:8080"
}

output "redis_private_ip" {
  description = "Private IP address of Redis instance"
  value       = aws_instance.redis.private_ip
}

output "worker_asg_name" {
  description = "Name of the Worker Auto Scaling Group (for scaling Experiment 2)"
  value       = aws_autoscaling_group.worker.name
}

output "instructions" {
  description = "Instructions for running experiments"
  value = <<-EOT
    ============================================
    Deployment Complete!
    ============================================
    
    API Endpoint: http://${aws_instance.api.public_ip}:8080
    
    To run experiments from your laptop:
    
    1. Update client code to use API endpoint:
       export API_ENDPOINT="http://${aws_instance.api.public_ip}:8080"
    
    2. Or modify client code:
       baseURL := "http://${aws_instance.api.public_ip}:8080"
    
    3. For Experiment 2 (worker scaling):
       aws autoscaling set-desired-capacity \
         --auto-scaling-group-name ${aws_autoscaling_group.worker.name} \
         --desired-capacity <N> \
         --region ${var.aws_region}
    
    4. Check worker instances:
       aws ec2 describe-instances \
         --filters "Name=tag:Name,Values=task-queue-worker" \
         --region ${var.aws_region} \
         --query 'Reservations[*].Instances[*].[InstanceId,State.Name,PrivateIpAddress]' \
         --output table
    
    ============================================
  EOT
}

