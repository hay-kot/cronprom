FROM gcr.io/distroless/static
ENTRYPOINT ["/cronprom", "serve"]
EXPOSE 8080
COPY cronprom /
