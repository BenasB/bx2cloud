FROM alpine:3
ARG BINARY_NAME=bx2cloud

COPY ${BINARY_NAME} /app

RUN printf "#!/bin/sh\nexec /app${BINARY_NAME}" > /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]