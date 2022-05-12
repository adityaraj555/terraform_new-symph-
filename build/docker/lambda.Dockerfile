FROM public.ecr.aws/lambda/go:1.2022.03.23.17

COPY bin/lambda /lambda

COPY build/assets/rds-combined-ca-bundle.pem /rds-combined-ca-bundle.pem

CMD ["./lambda"]