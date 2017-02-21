FROM centos:7

ADD ./build/gofabric8-linux-amd64 /bin/gofabric8

ENV ARGS version
ENV FABRIC8_BATCH true

ENTRYPOINT /bin/gofabric8 ${ARGS}