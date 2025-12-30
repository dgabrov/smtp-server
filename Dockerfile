FROM amd64/alpine
EXPOSE 8225
WORKDIR /app
COPY smtp-server-go .
CMD ["/app/smtp-server-go"]
