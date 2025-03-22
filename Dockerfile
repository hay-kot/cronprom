FROM gcr.io/distroless/static
ENTRYPOINT ["/cronprom", "run"]
COPY cronprom /
