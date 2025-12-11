# #!/bin/bash

# # get_api_ip.sh - Get API public IP from ECS task

# REGION="us-west-2"
# CLUSTER="task-queue-cluster"
# SERVICE="task-queue-api"

# echo "Getting API endpoint..."

# # Get task ARN
# TASK_ARN=$(aws ecs list-tasks \
#   --cluster $CLUSTER \
#   --service-name $SERVICE \
#   --region $REGION \
#   --query 'taskArns[0]' \
#   --output text)

# if [ -z "$TASK_ARN" ] || [ "$TASK_ARN" == "None" ]; then
#   echo "Error: No tasks found for API service"
#   echo "Make sure the service is running:"
#   echo "  aws ecs describe-services --cluster $CLUSTER --services $SERVICE --region $REGION"
#   exit 1
# fi

# echo "Task ARN: $TASK_ARN"

# # Get ENI ID from task
# ENI_ID=$(aws ecs describe-tasks \
#   --cluster $CLUSTER \
#   --tasks $TASK_ARN \
#   --region $REGION \
#   --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value' \
#   --output text)

# if [ -z "$ENI_ID" ]; then
#   echo "Error: Could not find network interface"
#   exit 1
# fi

# echo "Network Interface: $ENI_ID"

# # Get public IP from ENI
# PUBLIC_IP=$(aws ec2 describe-network-interfaces \
#   --network-interface-ids $ENI_ID \
#   --region $REGION \
#   --query 'NetworkInterfaces[0].Association.PublicIp' \
#   --output text)

# if [ -z "$PUBLIC_IP" ] || [ "$PUBLIC_IP" == "None" ]; then
#   echo "Error: No public IP assigned to task"
#   exit 1
# fi

# echo ""
# echo "✅ API Endpoint: http://$PUBLIC_IP:8080"
# echo ""
# echo "Test with:"
# echo "  curl -X POST http://$PUBLIC_IP:8080/task/fifo \\"
# echo "    -H 'Content-Type: application/json' \\"
# echo "    -d '{\"job_type\":\"short\"}'"
# echo ""

# # Export for use in other scripts
# export API_ENDPOINT="http://$PUBLIC_IP:8080"
# echo "Exported: API_ENDPOINT=$API_ENDPOINT"

cd terraform

# Get API IP from Terraform output
API_IP=$(terraform output -raw api_public_ip 2>/dev/null)

if [ -z "$API_IP" ]; then
    echo "Error: Could not get API IP from Terraform"
    echo "Make sure you've run 'terraform apply' first"
    exit 1
fi

echo ""
echo "✅ API Endpoint: http://$API_IP:8080"
echo ""
echo "Test with:"
echo "  curl -X POST http://$API_IP:8080/task/fifo \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"job_type\":\"short\"}'"
echo ""

export API_ENDPOINT="http://$API_IP:8080"
echo "Exported: API_ENDPOINT=$API_ENDPOINT"