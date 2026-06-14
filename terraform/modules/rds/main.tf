variable "name"            { type = string }
variable "vpc_id"          { type = string }
variable "private_subnets" { type = list(string) }
variable "ecs_sg_id"       { type = string }
variable "db_password"     {
  type      = string
  sensitive = true
}

resource "aws_security_group" "rds" {
  name        = "${var.name}-rds"
  description = "Allow Postgres from ECS only"
  vpc_id      = var.vpc_id

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [var.ecs_sg_id]
  }
}

resource "aws_db_subnet_group" "this" {
  name       = "${var.name}-rds"
  subnet_ids = var.private_subnets
}

resource "aws_db_instance" "this" {
  identifier             = "${var.name}-postgres"
  engine                 = "postgres"
  engine_version         = "16"
  instance_class         = "db.t3.micro"
  allocated_storage      = 20
  storage_type           = "gp3"
  storage_encrypted      = true
  db_name                = "appdb"
  username               = "appuser"
  password               = var.db_password
  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  multi_az               = false
  publicly_accessible    = false
  deletion_protection    = false
  skip_final_snapshot    = true
  backup_retention_period = 7
  tags = { Name = "${var.name}-postgres" }
}

resource "aws_secretsmanager_secret" "db" {
  name                    = "${var.name}/db-credentials"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "db" {
  secret_id = aws_secretsmanager_secret.db.id
  secret_string = jsonencode({
    username = aws_db_instance.this.username
    password = var.db_password
    host     = aws_db_instance.this.address
    port     = 5432
    dbname   = "appdb"
  })
}

output "db_host"    { value = aws_db_instance.this.address }
output "secret_arn" { value = aws_secretsmanager_secret.db.arn }
