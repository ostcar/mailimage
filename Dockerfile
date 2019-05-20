FROM golang as builder
LABEL maintainer="Oskar Hahn <mail@oshahn.de>"
WORKDIR /root/

# Copy everything from the current directory
COPY . .

# Build the Go app
RUN go generate && CGO_ENABLED=0 go build


######## Start a new stage from scratch #######
FROM scratch
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /root/mailimage .

EXPOSE 5000

CMD ["./mailimage", "serve"]