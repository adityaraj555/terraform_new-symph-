FROM public.ecr.aws/lambda/go:1.2022.03.23.17

COPY bin/main /main

COPY build/assets/rds-combined-ca-bundle.pem /rds-combined-ca-bundle.pem

COPY lambdas/querypdw/query.gql query.gql

ENTRYPOINT [ "/main" ]