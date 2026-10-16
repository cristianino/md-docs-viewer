# AWS Lambda (imagen de contenedor) con AWS Lambda Web Adapter.
# Corre "mdocs serve" sin cambios; el PDF y el .drawio se generan ANTES
# (mdocs build en CI) y se sirven en modo static, así no hace falta Chrome
# dentro de Lambda.
#
#   docker build -t mdocs:slim --target slim /ruta/a/md-docs-viewer
#   mdocs build                                   # en la carpeta de la documentación
#   docker build -f deploy/lambda/Dockerfile -t mis-docs-lambda .
#
# Contexto de build: la carpeta del proyecto de documentación.
FROM mdocs:slim
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.9.1 /lambda-adapter /opt/extensions/lambda-adapter
COPY . /work
ENV PORT=8080 \
    MDOCS_EXPORTS=static \
    MDOCS_DIST=dist \
    AWS_LWA_READINESS_CHECK_PATH=/healthz
