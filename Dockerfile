FROM gcr.io/distroless/static-debian12
COPY investor-linux /investor
ENTRYPOINT ["/investor"]
