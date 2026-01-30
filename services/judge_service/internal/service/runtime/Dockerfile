FROM golang:1.24-alpine

RUN apk add --no-cache python3 coreutils

COPY runner.sh /usr/local/bin/judge-runner

RUN chmod +x /usr/local/bin/judge-runner

CMD ["sh", "-c", "trap : TERM INT; sleep infinity & wait"]
