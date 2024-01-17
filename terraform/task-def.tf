resource "aws_ecs_task_definition" "mypointingpoker" {
  family                   = "mypointingpoker"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.ecs_execution_role.arn
  cpu                      = "256"
  memory                   = "512"
  container_definitions    = data.template_file.container_definitions.rendered
}

data "template_file" "container_definitions" {
  template = <<EOF
[
  {
    "name": "mypointingpoker",
    "image": "rdlucas2/pointingpoker:latest",
    "essential": true,
    "portMappings": [
      {
        "containerPort": 8080,
        "hostPort": 8080
      }
    ],
    "logConfiguration": {
      "logDriver": "awslogs",
      "options": {
        "awslogs-group": "${aws_cloudwatch_log_group.ecs_logs.name}",
        "awslogs-region": "${var.aws_region}",
        "awslogs-stream-prefix": "ecs"
      }
    }
  }
]
EOF
}