FROM mariadb:latest

COPY ./bin/mariadb-wrapper-agent /usr/local/bin/mariadb-wrapper-agent

#RUN cp ./mariadb-wrapper-agent /usr/local/bin/mariadb-wrapper-agent
RUN chmod +x /usr/local/bin/mariadb-wrapper-agent

#ENTRYPOINT ["/usr/local/bin/mariadb-wrapper-agent"]
CMD ["mariadb-wrapper-agent"]
