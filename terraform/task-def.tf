resource "aws_ecs_task_definition" "mypointingpoker" {
  family                   = "mypointingpoker"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.ecs_execution_role.arn
  cpu                      = "256"  # Adjust as needed
  memory                   = "512"  # Adjust as needed
  container_definitions    = jsonencode([{
    name  = "mypointingpoker"
    image = "rdlucas2/pointingpoker:latest"
    portMappings = [{
      containerPort = 8080
    }]
  }])
}