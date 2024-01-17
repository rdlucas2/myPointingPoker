resource "aws_ecs_service" "mypointingpoker_service" {
  name            = "mypointingpoker-service"
  cluster         = aws_ecs_cluster.cluster.id
  task_definition = aws_ecs_task_definition.mypointingpoker.arn
  #launch_type     = "FARGATE"
  desired_count   = 1 # Adjust based on your requirements

  capacity_provider_strategy {
    capacity_provider = "FARGATE_SPOT"
    weight            = 1
    base              = 0
  }

  network_configuration {
    subnets          = [aws_subnet.example.id] # Specify your subnet IDs
    security_groups  = [aws_security_group.example.id]
    assign_public_ip = true
  }

  deployment_controller {
    type = "ECS"
  }

#   load_balancer {
#     target_group_arn = aws_lb_target_group.my_tg.arn
#     container_name   = "mypointingpoker"
#     container_port   = 8080
#   }
}
