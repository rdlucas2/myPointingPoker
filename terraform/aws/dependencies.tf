data "aws_vpc" "default_vpc" {
  default = true
  #id = "vpc-00d5f93201daa4f25" # if not default, enter vpc's id
}

#use your vpc ids, you may wish to add more
data "aws_subnet" "default_subnet_a" {
  id = "subnet-00cc008097b50a86a"
}

data "aws_subnet" "default_subnet_b" {
  id = "subnet-0f9540cbdf49b8f9a"
}
