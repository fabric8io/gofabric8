FROM scratch

ADD ./build/gofabric8-linux-amd64 /bin/gofabric8

ENTRYPOINT ["/bin/gofabric8", "version"]