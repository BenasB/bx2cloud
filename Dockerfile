ARG BASE=alpine:3
FROM ${BASE}
ARG BINARY_NAME=bx2cloud

RUN apt-get update && apt-get install -y \
    iptables \
    ca-certificates

COPY ${BINARY_NAME} /app/${BINARY_NAME}

# Since ARG can't be substituted in ENTRYPOINT
RUN printf "#!/bin/sh\nexec /app/${BINARY_NAME}" > /entrypoint.sh \
    && chmod +x /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]
