resource "aws_iam_role" "cross-account-callback-lambda" {
  assume_role_policy = <<POLICY
{
   "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": [
                    "${module.config.environment_config_map.cross_account_callback_lambda}"
                ]
            },
            "Action": "sts:AssumeRole"
        }
    ]
}
POLICY

  inline_policy {
    name   = "callback-lambda-access-policy"
    policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "lambda:InvokeFunction",
                "lambda:InvokeAsync"
            ],
            "Resource": [
                "arn:aws:lambda:${local.region}:${local.account_id}:function:${local.resource_name_prefix}-lambda-${module.config.environment_config_map.callback_lambda_name}"
            ],
            "Effect": "Allow",
            "Sid": "AccessCallback"
        }
    ]
}

POLICY
  }

  managed_policy_arns = []

  max_session_duration = "3600"
  name                 = "${local.resource_name_prefix}-role-callback-lambda-access"
  path                 = "/"

  tags = {
    Name        = "${local.resource_name_prefix}-role-callback-lambda-access"
    Description = "AWS IAM role to allow cross account access from EV Factory account to callback lambda."
  }
}




resource "aws_iam_role" "platform-data-orchestrator-callback-lambda-s3" {
  assume_role_policy = module.config.environment_config_map.trust_relashionships_external_service

  inline_policy {
    name   = "platform-data-orchestrator-resources-access-policy"
    policy = module.config.environment_config_map.inline_policy_external_service
  }

  managed_policy_arns = []

  max_session_duration = "3600"
  name                 = "${local.resource_name_prefix}-role-pdo-access"
  path                 = "/"

  tags = {
    Name        = "${local.resource_name_prefix}-role-pdo-access"
    Description = "AWS IAM role to allow services to access platform-data-orchestrator common resources like s3 and callback Lambda"
  }
}
