data "aws_secretsmanager_secret" "secrets" {
  // In sandox, password is clear text, to avoid depending on an onboarding secret
  count = local.clear_text_db_password == null ? 1 : 0
  arn   = local.document_db_secret
}

data "aws_secretsmanager_secret_version" "secret" {
  // In sandox, password is clear text, to avoid depending on an onboarding secret
  count     = local.clear_text_db_password == null ? 1 : 0
  secret_id = data.aws_secretsmanager_secret.secrets[0].id
}
