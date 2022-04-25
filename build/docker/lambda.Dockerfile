FROM public.ecr.aws/lambda/go:1.2022.03.23.17

COPY bin/lambda /lambda

CMD ["./lambda"]