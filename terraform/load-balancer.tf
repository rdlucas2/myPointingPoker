resource "aws_lb" "my_alb" {
  name               = "mypointingpoker-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb_sg.id]
  subnets            = [aws_subnet.example.id]  # Replace with your subnet IDs

  enable_deletion_protection = false
}

resource "aws_security_group" "alb_sg" {
  name        = "mypointingpoker-alb-sg"
  description = "ALB security group"
  vpc_id      = aws_vpc.example.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_lb_target_group" "my_tg" {
  name     = "mypointingpoker-tg"
  port     = 8080
  protocol = "HTTP"
  vpc_id   = aws_vpc.example.id

  health_check {
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 5
    path                = "/"  # Modify if you have a specific health check endpoint
    interval            = 30
    matcher             = "200"
  }
}

resource "aws_lb_listener" "my_listener" {
  load_balancer_arn = aws_lb.my_alb.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.my_tg.arn
  }
}
