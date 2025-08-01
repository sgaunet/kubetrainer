FROM alpine:3.22.1 AS alpine

FROM scratch AS final
WORKDIR /usr/local/bin
COPY kubetrainer .
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY "resources" /
USER kubetrainer

EXPOSE 3000
CMD ["kubetrainer"]
