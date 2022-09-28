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


resource "aws_iam_role" "platform-data-orchestrator-service-account-to-pdo-role" {
 assume_role_policy = <<POLICY
${module.config.environment_config_map.trust_relashionships_external_service_factory_dx}
  POLICY

  inline_policy {
    name   = "platform-data-orchestrator-service-account-to-pdo-policy"
    policy = <<POLICY
{
        "Version": "2012-10-17",
        "Statement": [
            {
                "Action": [
                    "lambda:InvokeFunction",
                    "lambda:InvokeAsync",
                    "ec2:DescribeInstances",
                    "ec2:DescribeInstanceStatus",
                    "ec2:DeleteTags",
                    "ec2:CreateTags",
                    "s3:PutObject",
                    "s3:PutObjectAcl",
                    "s3:DeleteObject",
                    "s3:GetObject",
                    "s3:GetObjectAcl",
                    "sqs:DeleteMessage",
                    "sqs:ReceiveMessage",
                    "sqs:GetQueueAttributes",
                    "sqs:SendMessage"
                ],
                "Resource": [
                    "arn:aws:lambda:${local.region}:${local.account_id}:function:${local.resource_name_prefix}-lambda-${module.config.environment_config_map.callback_lambda_name}",
                    "arn:aws:s3:::${local.resource_name_prefix}-s3-property-data-orchestrator",
                    "arn:aws:s3:::${local.resource_name_prefix}-s3-property-data-orchestrator/*",
                    "arn:aws:sqs:${local.region}:${local.account_id}:${local.resource_name_prefix}-sqs-receiveLegacyOrder",
                    "arn:aws:sqs:${local.region}:${local.account_id}:${local.resource_name_prefix}-sqs-receiveSIMOrder"

                ],
                "Effect": "Allow",
                "Sid": "AccessCallback2"
            },
            {
              "Effect": "Allow",
              "Action": [
                "states:DescribeStateMachine",
                "states:StartExecution",
                "states:DeleteStateMachine",
                "states:ListExecutions",
                "states:UpdateStateMachine"
              ],
              "Resource": [ 
                "arn:aws:states:${local.region}:${local.account_id}:stateMachine:${local.resource_name_prefix}-sfn-evoss-rerun-sfn",
                "arn:aws:states:${local.region}:${local.account_id}:stateMachine:${local.resource_name_prefix}-sfn-sim",
                "arn:aws:states:${local.region}:${local.account_id}:stateMachine:${local.resource_name_prefix}-sfn-ais",
                "arn:aws:states:${local.region}:${local.account_id}:stateMachine:${local.resource_name_prefix}-sfn-symphony_workflow"
              ]
            }

        ]
    }
    POLICY
    
  }

  managed_policy_arns = []

  max_session_duration = "3600"
  name                 = "${local.resource_name_prefix}-service-account-to-pdo"
  path                 = "/"

  tags = {
    Name        = "${local.resource_name_prefix}-service-account-to-pdo"
    Description = "AWS IAM role to allow services to access platform-data-orchestrator common resources like s3 and callback Lambda"
  }
}

resource "aws_iam_role" "platform-data-orchestrator-callback-lambda-s3" {
  assume_role_policy = <<POLICY
${module.config.environment_config_map.trust_relashionships_external_service}
  POLICY

  inline_policy {
    name   = "platform-data-orchestrator-resources-access-policy"
    policy = <<POLICY
{
        "Version": "2012-10-17",
        "Statement": [
            {
                "Action": [
                    "lambda:InvokeFunction",
                    "lambda:InvokeAsync",
                    "ec2:DescribeInstances",
                    "ec2:DescribeInstanceStatus",
                    "ec2:DeleteTags",
                    "ec2:CreateTags",
                    "s3:PutObject",
                    "s3:PutObjectAcl",
                    "s3:DeleteObject",
                    "s3:GetObject",
                    "s3:GetObjectAcl"
                ],
                "Resource": [
                    "arn:aws:lambda:${local.region}:${local.account_id}:function:${local.resource_name_prefix}-lambda-${module.config.environment_config_map.callback_lambda_name}",
                    "arn:aws:s3:::${local.resource_name_prefix}-s3-property-data-orchestrator",
                    "arn:aws:s3:::${local.resource_name_prefix}-s3-property-data-orchestrator/*"

                ],
                "Effect": "Allow",
                "Sid": "AccessCallback2"
            }
        ]
    }
    POLICY
    
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
