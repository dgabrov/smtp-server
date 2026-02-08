FROM amd64/alpine
EXPOSE 8225
EXPOSE 8587
EXPOSE 8993
RUN apk add --no-cache ca-certificates && update-ca-certificates
WORKDIR /app
COPY smtp-server-go .
CMD ["/app/smtp-server-go"]
