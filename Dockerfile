#intended to be used for local development by mounting pwd as volume
FROM golang:1.21.3 AS local
ENTRYPOINT ["/bin/bash"]

FROM golang:1.21.3-alpine AS base
RUN apk add --update gcc musl-dev
WORKDIR /app
COPY go.mod go.mod
COPY go.sum go.sum
RUN go get github.com/mattn/go-sqlite3
COPY ./src /app/src/
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o myapp /app/src

FROM base AS test
RUN go install gotest.tools/gotestsum@latest
ENTRYPOINT ["gotestsum", "--jsonfile", "/out/tests.json", "--", "-coverprofile=/out/coverage.out", "./..."]

FROM base AS run
WORKDIR /app/src
ENTRYPOINT [ "go", "run", "." ]

# Final stage
FROM alpine AS artifact
RUN rm -f /sbin/apk && \
    rm -rf /etc/apk && \
    rm -rf /lib/apk && \
    rm -rf /usr/share/apk && \
    rm -rf /var/lib/apk

COPY --from=base /app/myapp /home/nonroot/gohtmx
COPY --from=base /app/src/templates /home/nonroot/templates
COPY --from=base /app/src/static /home/nonroot/static
EXPOSE 3000
RUN addgroup --system nonroot && \
    adduser --system --ingroup nonroot nonroot
# Set the home directory for the nonroot user
ENV HOME=/home/nonroot
# Create the home directory and set proper permissions
RUN chown -R nonroot:nonroot $HOME && \
    chown -R nonroot:nonroot /tmp
# Switch to the nonroot user
USER nonroot
WORKDIR /home/nonroot
ENTRYPOINT ["./gohtmx"]
