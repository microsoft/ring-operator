FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/ring-operator \
    USER_UID=1001 \
    USER_NAME=ring-operator

# install operator binary
COPY build/_output/bin/ring-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  chown ${USER_UID}:${USER_UID} -R /usr/local/bin && \
     /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
