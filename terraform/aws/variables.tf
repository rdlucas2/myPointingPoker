variable "aws_region" {
  type = string
  default = "us-east-2"
}

variable "app_name" {
  type = string
  default = "pointing-poker"
}

variable "image_name" {
  type = string
  default = "rdlucas2/pointingpoker:latest"
}

variable "container_port" {
  type = number
  default = 8080
}

variable "memory" {
  type = number
  default = 512
}

variable "cpu" {
  type = number
  default = 256
}
