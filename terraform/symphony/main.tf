
module "config" {
  source = "git::https://github.eagleview.com/engineering/symphony-config.git//${platform_choice}${test_config_branch}"
}

module "lambda" {
  source = "git::https://github.eagleview.com/infrastructure/terraform-cloudops-module-lambda.git//lambda?ref=3.1.0"

  providers = {
    aws = aws
  }

  for_each = module.config.lambda_configmap

  resource_name_prefix  = local.resource_name_prefix
  image_uri             = try(each.value.image_uri, null)
  package_type          = try(each.value.package_type, "Image")
  vpc_id                = each.value.vpc_id
  environment_variables = try(each.value.environment_variables, null)
  lambda_name           = each.key
  lambda_handler        = each.value.lambda_handler
  lambda_description    = each.value.lambda_description
  managed_policy_arns   = each.value.managed_policy_arns
  lambda_inline_policy  = try(each.value.lambda_inline_policy, null)
  schedule_time_trigger = try(each.value.schedule_time_trigger, null)
  aws_lambda_permission = try(each.value.aws_lambda_permission, [])
  lambda_assume_role_policy = try(each.value.lambda_assume_role_policy, null)
  timeout               = try(each.value.timeout, 3)
  memory_size           = try(each.value.memory_size, 128)
  source_path           = null
}

module "invokesfn_lambda" {
  source = "git::https://github.eagleview.com/infrastructure/terraform-cloudops-module-lambda.git//lambda?ref=3.1.0"

  providers = {
    aws = aws
  }

  for_each = module.config.sfn_lambda_configmap

  resource_name_prefix      = local.resource_name_prefix
  image_uri                 = try(each.value.image_uri, null)
  package_type              = try(each.value.package_type, "Image")
  vpc_id                    = each.value.vpc_id
  environment_variables     = try(each.value.environment_variables, null)
  lambda_name               = each.key
  lambda_handler            = each.value.lambda_handler
  lambda_description        = each.value.lambda_description
  managed_policy_arns       = each.value.managed_policy_arns
  lambda_inline_policy      = try(each.value.lambda_inline_policy, null)
  lambda_assume_role_policy = try(each.value.lambda_assume_role_policy, null)
  schedule_time_trigger     = try(each.value.schedule_time_trigger, null)
  aws_lambda_permission     = try(each.value.aws_lambda_permission, [])
  timeout                   = try(each.value.timeout, 3)
  memory_size               = try(each.value.memory_size, 128)
  source_path               = null
}


module "step_function" {
  source = "git::https://github.eagleview.com/infrastructure/terraform-cloudops-module-step-function.git//step_function/?ref=2.0.0"
  providers = {
    aws = aws
  }
  for_each = module.config.step_function_config_map

  sfn_name             = each.key
  resource_name_prefix = local.resource_name_prefix
  source_path          = each.value.source_path
  sfn_def_env_vars     = each.value.sfn_def_env_vars
  depends_on           = [module.lambda]
}

module "s3" {
  source = "git::https://github.eagleview.com/infrastructure/terraform-cloudops-module-s3-bucket.git//s3/?ref=2.0.0"
  providers = {
    aws = aws
  }

  for_each = module.config.s3_config_map

  s3_name              = each.key
  identifiers          = each.value.identifiers
  actions              = each.value.actions
  resource_name_prefix = local.resource_name_prefix
}

module "documentdb" {
  source = "git::https://github.eagleview.com/infrastructure/terraform-cloudops-module-documentDB//documentdb/?ref=1.1.0"
  providers = {
    aws = aws
  }

  vpc_id                       = module.config.document_db_config_map.vpc_id
  documentdb_name              = module.config.document_db_config_map.name
  resource_name_prefix         = local.resource_name_prefix
  db_port                      = module.config.document_db_config_map.db_port
  master_username              = module.config.document_db_config_map.master_username
  master_password              = local.document_db_password
  retention_period             = module.config.document_db_config_map.retention_period
  preferred_backup_window      = module.config.document_db_config_map.preferred_backup_window
  preferred_maintenance_window = module.config.document_db_config_map.preferred_maintenance_window
  skip_final_snapshot          = module.config.document_db_config_map.skip_final_snapshot
  deletion_protection          = module.config.document_db_config_map.deletion_protection
  apply_immediately            = module.config.document_db_config_map.apply_immediately
  engine                       = module.config.document_db_config_map.engine
  engine_version               = module.config.document_db_config_map.engine_version
  subnet_ids                   = module.config.document_db_config_map.subnet_ids
  cluster_size                 = module.config.document_db_config_map.cluster_size
  instance_class               = module.config.document_db_config_map.instance_class
  cluster_family               = module.config.document_db_config_map.cluster_family
  allowed_cidr_blocks          = module.config.document_db_config_map.allowed_cidr_blocks

}

resource "aws_sqs_queue" "sqs" {

  for_each = module.config.sqs_config_map

  name                       = "${local.resource_name_prefix}-sqs-${each.key}"
  delay_seconds              = each.value.delay_seconds
  max_message_size           = each.value.max_message_size
  message_retention_seconds  = each.value.message_retention_seconds
  receive_wait_time_seconds  = each.value.receive_wait_time_seconds
  policy                     = each.value.policy
  visibility_timeout_seconds = each.value.visibility_timeout_seconds
}

// Name of the legacy order queue
resource "aws_lambda_event_source_mapping" "event_trigger_sqs" {
  event_source_arn = "arn:aws:sqs:${local.region}:${local.account_id}:${local.resource_name_prefix}-sqs-${module.config.environment_config_map.receive_legacy_order_queue_name}"
  function_name    = "arn:aws:lambda:${local.region}:${local.resource_name_prefix}-lambda-${module.config.environment_config_map.invokesfn_lambda_name}" //module.invokesfn_lambda[0].arn
  depends_on       = [module.invokesfn_lambda]
}
//

resource "aws_sns_topic_subscription" "lambda_sns_subscription" {
      topic_arn = "arn:aws:sns:us-east-2:356071200662:DomainEvents"
      protocol  = "sqs"
      endpoint  = "arn:aws:sqs:us-east-2:356071200662:app-dev-1x0-sqs-receiveLegacyOrder"
      filter_policy = "${jsonencode(map("Company",list("eagleview")))}"
      raw_message_delivery = true
}

data "aws_caller_identity" "current" {}

// Useful to troubleshoot role issues
output "assumed-identity-arn" {
  value = data.aws_caller_identity.current.arn
}
